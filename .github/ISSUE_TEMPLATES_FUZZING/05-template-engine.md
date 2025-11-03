---
title: "Expand fuzzy testing coverage for template engine"
labels: ["enhancement", "testing", "templates"]
---

## Description

Expand the existing fuzzy testing coverage for the template engine (`internal/tmpl`) to cover more edge cases and security scenarios.

## Rationale

The template engine:
- Already has some fuzzy tests in `internal/tmpl/fuzz_test.go`
- Is used extensively throughout GoReleaser for string interpolation
- Template injection vulnerabilities are a real security concern
- Handles user-provided input in many contexts
- Needs expanded coverage for additional template functions and edge cases

## Current Coverage

Existing tests in `internal/tmpl/fuzz_test.go`:
- `FuzzTemplateApplier`
- `FuzzTemplateWithArtifact`
- `FuzzTemplateBool`
- `FuzzTemplateSlice`
- `FuzzTemplateWithBuildOptions`

## Proposed Enhancements

### 1. Add `FuzzTemplateWithEnv`
Test template evaluation with environment variables:
- Very long environment variable values
- Special characters in env vars
- Unicode in environment variables
- Null bytes and control characters

### 2. Add `FuzzTemplateWithExtraFields`
Test custom field handling:
- Deeply nested field access
- Invalid field names
- Non-existent field references
- Type mismatches

### 3. Add `FuzzTemplateNestedExpressions`
Test complex nested templates:
- Multiple levels of nesting
- Mixed functions
- Control structures (if/range)
- Edge cases in conditionals

### 4. Add `FuzzTemplateFunctions`
Test individual template functions:
- String manipulation functions
- Path operations
- Date/time formatting
- Custom Sprig functions

### 5. Expand existing tests
Add more seed inputs covering:
- ReDoS patterns in regex functions
- Path traversal attempts
- Code injection attempts
- Very long template strings (DoS)

## Example Enhancement

```go
func FuzzTemplateWithEnv(f *testing.F) {
    f.Add("{{ .Env.MY_VAR }}", "normal_value")
    f.Add("{{ .Env.PATH }}", "/usr/bin:/bin")
    f.Add("{{ .Env.SPECIAL }}", "value;with;special;chars")
    
    f.Fuzz(func(t *testing.T, template, envValue string) {
        ctx := testctx.New()
        ctx.Env = map[string]string{"MY_VAR": envValue, "PATH": envValue, "SPECIAL": envValue}
        
        tpl := New(ctx)
        _, err := tpl.Apply(template)
        if err == nil {
            return
        }
        require.ErrorAs(t, err, &Error{})
    })
}
```

## Acceptance Criteria

- [ ] Add at least 4 new fuzz test functions
- [ ] Expand seed corpus in existing tests
- [ ] Add tests for all major template function categories
- [ ] Test security-sensitive scenarios (injection, traversal, DoS)
- [ ] Document any discovered edge cases
- [ ] Ensure tests run efficiently in CI

## Related Files

- `internal/tmpl/tmpl.go`
- `internal/tmpl/fuzz_test.go` (existing)
- `internal/tmpl/errors.go`

## Priority

**High** - Template engine is security-critical and used throughout the codebase.
