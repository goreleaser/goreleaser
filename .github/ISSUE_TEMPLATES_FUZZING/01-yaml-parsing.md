---
title: "Add fuzzy testing for YAML configuration parsing"
labels: ["enhancement", "testing", "security"]
---

## Description

Add comprehensive fuzzy testing for the YAML configuration parsing module (`internal/yaml`) to improve robustness and security.

## Rationale

The YAML parser handles GoReleaser's primary configuration files and is critical for security and stability:
- Malformed YAML could cause crashes or unexpected behavior
- Both strict and non-strict unmarshaling modes need testing
- Handles user-provided configuration files
- Currently no fuzzy tests exist for this module

## Proposed Implementation

Create `internal/yaml/yaml_fuzz_test.go` with the following tests:

### 1. `FuzzUnmarshalStrict`
Test strict YAML unmarshaling with various inputs:
- Malformed YAML structures
- Deeply nested structures
- Very large files
- Unicode and special characters
- Invalid field names
- Type mismatches

### 2. `FuzzUnmarshal`
Test non-strict YAML unmarshaling:
- Extra fields
- Missing fields
- Mixed valid/invalid content
- Edge cases in type coercion

### 3. `FuzzMarshal`
Test YAML marshaling:
- Complex nested structures
- Special characters
- Very large data structures
- Null/empty values

## Example Test Structure

```go
func FuzzUnmarshalStrict(f *testing.F) {
    // Add seed corpus
    f.Add([]byte("version: 2\nproject_name: test"))
    f.Add([]byte("version: 2\n\ninvalid"))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        var out config.Project
        err := UnmarshalStrict(data, &out)
        // Should not panic
        _ = err
    })
}
```

## Acceptance Criteria

- [ ] Create `internal/yaml/yaml_fuzz_test.go`
- [ ] Implement at least 3 fuzz test functions
- [ ] Add seed corpus with known edge cases
- [ ] Tests should run without panics
- [ ] Integrate with existing test suite
- [ ] Update CI to run fuzzing tests

## Related Files

- `internal/yaml/yaml.go`
- `internal/yaml/yaml_test.go`
- `scripts/fuzz.sh`

## Priority

**Critical** - YAML parsing is the entry point for all GoReleaser configurations.
