**Title:** Add tests for NFPM conversion methods in pkg/config/nfpm.go

**Labels:** good first issue, test coverage, enhancement

**Description:**

The `pkg/config/nfpm.go` file contains 2 conversion methods that currently have 0% test coverage. These methods convert GoReleaser config types to NFPM library types.

### Affected Methods
- `NFPMIPKAlternative.ToNFP()` (line 5)
- `NFPMIPK.ToNFPAlts()` (line 13)

### Current Coverage
**0%** - Neither method is currently tested.

### What needs to be done
1. Create a new test file `pkg/config/nfpm_test.go`
2. Add test cases to verify the conversion logic
3. Ensure all fields are properly mapped
4. Test with both empty and populated structures

### Expected Impact
This will further improve coverage in `pkg/config` from ~46% to ~48%.

### Example Test Structure
```go
func TestNFPMIPKAlternativeToNFP(t *testing.T) {
    alt := NFPMIPKAlternative{
        Priority: 100,
        Target:   "/usr/bin/app",
        LinkName: "app",
    }
    
    result := alt.ToNFP()
    require.Equal(t, 100, result.Priority)
    require.Equal(t, "/usr/bin/app", result.Target)
    require.Equal(t, "app", result.LinkName)
}

func TestNFPMIPKToNFPAlts(t *testing.T) {
    ipk := NFPMIPK{
        Alternatives: []NFPMIPKAlternative{
            {Priority: 100, Target: "/usr/bin/app1", LinkName: "app1"},
            {Priority: 200, Target: "/usr/bin/app2", LinkName: "app2"},
        },
    }
    
    result := ipk.ToNFPAlts()
    require.Len(t, result, 2)
    require.Equal(t, 100, result[0].Priority)
    require.Equal(t, "/usr/bin/app1", result[0].Target)
}
```

### Files to modify
- Create: `pkg/config/nfpm_test.go`

### Difficulty
**Easy** - Simple conversion methods that just need to verify field mapping.
