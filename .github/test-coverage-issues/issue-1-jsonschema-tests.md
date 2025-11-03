**Title:** Add tests for JSONSchema methods in pkg/config

**Labels:** good first issue, test coverage, enhancement

**Description:**

The `pkg/config/jsonschema.go` file contains 8 JSONSchema methods that currently have 0% test coverage. These methods are straightforward and would be easy to test.

### Affected Methods
- `Hook.JSONSchema()` (line 5)
- `File.JSONSchema()` (line 21)
- `Hooks.JSONSchema()` (line 37)
- `FlagArray.JSONSchema()` (line 53)
- `StringArray.JSONSchema()` (line 66)
- `NixDependency.JSONSchema()` (line 79)
- `PullRequestBase.JSONSchema()` (line 95)
- `HomebrewDependency.JSONSchema()` (line 111)

### Current Coverage
**0%** - None of these methods are currently tested.

### What needs to be done
1. Create a new test file `pkg/config/jsonschema_test.go`
2. Add test cases for each JSONSchema method
3. Verify that each method returns a valid JSONSchema with the expected structure
4. Test both the string and object/array forms where applicable (OneOf schemas)

### Expected Impact
This will improve coverage in `pkg/config` from ~46% to approximately 55-60%.

### Example Test Structure
```go
func TestHookJSONSchema(t *testing.T) {
    schema := Hook{}.JSONSchema()
    require.NotNil(t, schema)
    require.NotNil(t, schema.OneOf)
    require.Len(t, schema.OneOf, 2)
    // Verify one option is string type
    // Verify one option is the expanded struct
}
```

### Files to modify
- Create: `pkg/config/jsonschema_test.go`

### Difficulty
**Easy** - These are straightforward schema generation methods that just need basic validation tests.
