---
title: "Add fuzzy testing for config loading"
labels: ["enhancement", "testing", "config", "security"]
---

## Description

Add comprehensive fuzzy testing for the configuration loading module (`pkg/config`) to improve robustness and security of GoReleaser's primary configuration system.

## Rationale

The config loading module:
- Is the main entry point for loading GoReleaser configurations
- Handles version detection and validation
- Has a very large and complex configuration structure (1400+ lines)
- Errors here affect all downstream operations
- Currently no fuzzy tests exist for this critical module

## Proposed Implementation

Create `pkg/config/config_fuzz_test.go` with the following tests:

### 1. `FuzzLoadReader`
Test configuration loading from various inputs:
- Malformed YAML
- Mixed valid/invalid sections
- Very large configuration files
- Deeply nested structures
- Invalid version numbers
- Pro config detection edge cases

### 2. `FuzzVersionValidation`
Test version detection and validation:
- Invalid version values
- Missing version field
- Version as string vs. number
- Very large version numbers

### 3. `FuzzConfigMarshaling`
Test configuration marshaling/unmarshaling round trips:
- Complex nested configurations
- All field types
- Default values
- Optional vs required fields

### 4. `FuzzConfigValidation`
Test configuration validation logic:
- Conflicting settings
- Invalid field combinations
- Missing required fields
- Out-of-range values

## Example Test Structure

```go
func FuzzLoadReader(f *testing.F) {
    // Add seed corpus with various config snippets
    f.Add([]byte("version: 2\nproject_name: test"))
    f.Add([]byte("version: 1\nproject_name: old"))
    f.Add([]byte("pro: true\nversion: 2"))
    f.Add([]byte("invalid: {{{{ yaml"))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        _, err := LoadReader(bytes.NewReader(data))
        // Should not panic, but may return error
        if err != nil {
            // Verify error is properly typed
            var verr VersionError
            if errors.As(err, &verr) {
                return
            }
            if errors.Is(err, ErrProConfig) {
                return
            }
            // Other errors are also acceptable
            return
        }
    })
}
```

## Acceptance Criteria

- [ ] Create `pkg/config/config_fuzz_test.go`
- [ ] Implement at least 4 fuzz test functions
- [ ] Add comprehensive seed corpus
- [ ] Tests should not panic on any input
- [ ] Verify proper error handling and typing
- [ ] Test all major config sections
- [ ] Integrate with existing test suite

## Related Files

- `pkg/config/load.go`
- `pkg/config/config.go`
- `pkg/config/config_test.go`
- `internal/yaml/yaml.go`

## Priority

**Critical** - Config loading is the entry point for all GoReleaser operations.
