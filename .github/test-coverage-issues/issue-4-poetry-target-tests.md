**Title:** Add tests for Target methods in internal/builders/poetry

**Labels:** good first issue, test coverage, enhancement

**Description:**

The `internal/builders/poetry/build.go` file contains a `Target` type with two methods that have 0% test coverage. The same pattern also exists in `internal/builders/uv`.

### Affected Methods
- `Target.Fields()` (line 62 in build.go)
- `Target.String()` (line 70 in build.go)

### Current Coverage
- `internal/builders/poetry`: **39.6%** → target **~50%**
- `internal/builders/uv`: **39.6%** → target **~50%** (same pattern)

### What needs to be done
1. Add test cases in `internal/builders/poetry/build_test.go`
2. Test that `Target.Fields()` returns the expected map with "all" values
3. Test that `Target.String()` returns "none-any"
4. Optionally apply the same tests to `internal/builders/uv/build_test.go`

### Example Test Structure
```go
func TestTargetFields(t *testing.T) {
    target := Target{}
    fields := target.Fields()
    require.Equal(t, map[string]string{
        tmpl.KeyOS:   "all",
        tmpl.KeyArch: "all",
    }, fields)
}

func TestTargetString(t *testing.T) {
    target := Target{}
    require.Equal(t, "none-any", target.String())
}
```

### Files to modify
- Modify: `internal/builders/poetry/build_test.go`
- Optional: Also add to `internal/builders/uv/build_test.go` (same pattern)

### Difficulty
**Very Easy** - These are trivial getter methods that just need basic assertion tests.
