**Title:** Add tests for Env.Copy() and Env.Strings() methods in pkg/context

**Labels:** good first issue, test coverage, enhancement

**Description:**

The `pkg/context/context.go` file contains two utility methods on the `Env` type that currently have 0% test coverage.

### Affected Methods
- `Env.Copy()` (line 43)
- `Env.Strings()` (line 51)

### Current Coverage
**57.9%** → target **~80%**

### What needs to be done
1. Add test cases in `pkg/context/context_test.go`
2. Test that `Env.Copy()` creates an independent copy (not a reference)
3. Test that `Env.Strings()` returns properly formatted "KEY=VALUE" strings
4. Test edge cases (empty env, special characters in values, etc.)

### Example Test Structure
```go
func TestEnvCopy(t *testing.T) {
    original := Env{"FOO": "bar", "BAZ": "qux"}
    copied := original.Copy()
    
    // Verify copy has same values
    require.Equal(t, original, copied)
    
    // Verify it's a separate copy (modifications don't affect original)
    copied["FOO"] = "modified"
    require.Equal(t, "bar", original["FOO"])
    require.Equal(t, "modified", copied["FOO"])
}

func TestEnvStrings(t *testing.T) {
    env := Env{"FOO": "bar", "BAZ": "qux"}
    strs := env.Strings()
    
    require.Len(t, strs, 2)
    require.Contains(t, strs, "FOO=bar")
    require.Contains(t, strs, "BAZ=qux")
}

func TestEnvStringsEmpty(t *testing.T) {
    env := Env{}
    strs := env.Strings()
    require.Empty(t, strs)
}
```

### Files to modify
- Modify: `pkg/context/context_test.go`

### Difficulty
**Very Easy** - Simple utility methods with straightforward testing requirements.
