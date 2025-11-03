---
title: "Add fuzzy testing for pyproject.toml parsing"
labels: ["enhancement", "testing", "python"]
---

## Description

Add fuzzy testing for the pyproject.toml parsing module (`internal/pyproject`) to improve robustness when processing Python project configuration files.

## Rationale

The pyproject.toml parser:
- Parses Python project configuration files
- Poetry detection relies on correct parsing
- TOML parsing errors could break Python project builds
- Handles external input from user projects

## Proposed Implementation

Create `internal/pyproject/pyproject_fuzz_test.go` with the following tests:

### 1. `FuzzOpen`
Test pyproject.toml file parsing:
- Malformed TOML structures
- Invalid project names
- Edge cases in version strings
- Malformed tool.poetry sections
- Missing required fields
- Very large files

### 2. `FuzzIsPoetry`
Test Poetry project detection:
- Edge cases in package configurations
- Empty package lists
- Malformed package entries

### 3. `FuzzName`
Test project name extraction and transformation:
- Special characters in names
- Very long names
- Unicode characters
- Names with multiple hyphens

## Example Test Structure

```go
func FuzzOpen(f *testing.F) {
    f.Add([]byte(`[project]
name = "test-project"
version = "1.0.0"`))
    f.Add([]byte(`[tool.poetry]
packages = [{include = "mypackage"}]`))
    f.Add([]byte(`invalid toml`))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        tmpfile := filepath.Join(t.TempDir(), "pyproject.toml")
        require.NoError(t, os.WriteFile(tmpfile, data, 0o644))
        
        proj, err := Open(tmpfile)
        if err != nil {
            return // Expected for invalid TOML
        }
        
        // Should not panic
        _ = proj.IsPoetry()
        _ = proj.Name()
    })
}
```

## Acceptance Criteria

- [ ] Create `internal/pyproject/pyproject_fuzz_test.go`
- [ ] Implement at least 3 fuzz test functions
- [ ] Add seed corpus with valid and invalid pyproject.toml examples
- [ ] Tests handle malformed TOML gracefully
- [ ] Verify name transformation handles edge cases
- [ ] Integrate with existing test suite

## Related Files

- `internal/pyproject/pyproject.go`
- `internal/pyproject/pyproject_test.go`
- `internal/pyproject/testdata/`

## Priority

**High** - Pyproject.toml parsing is critical for Python project support.
