---
title: "Add fuzzy testing for package.json parsing"
labels: ["enhancement", "testing", "nodejs", "bun"]
---

## Description

Add fuzzy testing for the package.json parsing module (`internal/packagejson`) to improve robustness when processing Node.js and Bun project files.

## Rationale

The package.json parser:
- Handles external JSON files from Node.js/Bun projects
- Detection of Bun projects depends on parsing accuracy
- JSON parsing vulnerabilities could lead to security issues
- Malformed JSON could break builds

## Proposed Implementation

Create `internal/packagejson/packagejson_fuzz_test.go` with the following tests:

### 1. `FuzzOpen`
Test package.json file parsing:
- Malformed JSON structures
- Invalid UTF-8 sequences
- Deeply nested dependencies
- Very large files
- Missing required fields
- Invalid field types

### 2. `FuzzIsBun`
Test Bun detection logic:
- Edge cases in devDependencies
- Malformed dependency objects
- Special characters in dependency names

## Example Test Structure

```go
func FuzzOpen(f *testing.F) {
    f.Add([]byte(`{"name": "test", "version": "1.0.0"}`))
    f.Add([]byte(`{"name": "test", "devDependencies": {"@types/bun": "^1.0.0"}}`))
    f.Add([]byte(`{invalid json`))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        tmpfile := filepath.Join(t.TempDir(), "package.json")
        require.NoError(t, os.WriteFile(tmpfile, data, 0o644))
        
        pkg, err := Open(tmpfile)
        if err != nil {
            return // Expected for invalid JSON
        }
        
        // Should not panic when checking IsBun
        _ = pkg.IsBun()
    })
}
```

## Acceptance Criteria

- [ ] Create `internal/packagejson/packagejson_fuzz_test.go`
- [ ] Implement at least 2 fuzz test functions
- [ ] Add seed corpus with valid and invalid package.json examples
- [ ] Tests handle malformed JSON gracefully
- [ ] Verify IsBun() doesn't panic on edge cases
- [ ] Integrate with existing test suite

## Related Files

- `internal/packagejson/packagejson.go`
- `internal/packagejson/packagejson_test.go`
- `internal/packagejson/testdata/`

## Priority

**High** - Package.json parsing is critical for Node.js and Bun build support.
