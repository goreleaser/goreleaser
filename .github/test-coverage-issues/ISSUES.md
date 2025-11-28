# GitHub Issues to Create for Test Coverage Improvements

Based on the test coverage analysis, here are the issues that should be created to improve test coverage. Each issue represents an easily addressable area with low coverage.

---

## Issue 1: Add tests for JSONSchema methods in pkg/config

**Title:** Add tests for JSONSchema methods in pkg/config

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
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
```

---

## Issue 2: Add tests for NFPM conversion methods in pkg/config

**Title:** Add tests for NFPM conversion methods in pkg/config/nfpm.go

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
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
```

### Files to modify
- Create: `pkg/config/nfpm_test.go`

### Difficulty
**Easy** - Simple conversion methods that just need to verify field mapping.
```

---

## Issue 3: Add tests for Copy methods in archive packages

**Title:** Add tests for Copy methods in pkg/archive/zip and pkg/archive/targz

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
Both `pkg/archive/zip/zip.go` and `pkg/archive/targz/targz.go` have `Copy` methods with 0% test coverage. There's even a TODO comment in `zip_test.go` line 169 that says "TODO: add copying test".

### Affected Methods
- `zip.Copy()` in `pkg/archive/zip/zip.go` (line 35)
- `targz.Copy()` in `pkg/archive/targz/targz.go` (line 30)

### Current Coverage
- `pkg/archive/zip`: **54.8%** → target **~75%**
- `pkg/archive/targz`: **46.2%** → target **~70%**

### What needs to be done
1. Add test for `zip.Copy()` in `pkg/archive/zip/zip_test.go`
2. Add test for `targz.Copy()` in `pkg/archive/targz/targz_test.go` (if it exists)
3. Tests should:
   - Create a source archive with known contents
   - Copy it to a new archive
   - Verify the copied archive contains the same files
   - Verify file permissions and metadata are preserved

### Example Test Structure
```go
func TestZipCopy(t *testing.T) {
    // Create source zip with test files
    var sourceBuf bytes.Buffer
    sourceArchive := New(&sourceBuf)
    sourceArchive.Add(config.File{...})
    sourceArchive.Close()
    
    // Copy to new archive
    var targetBuf bytes.Buffer
    sourceReader := bytes.NewReader(sourceBuf.Bytes())
    copiedArchive, err := Copy(sourceReader, &targetBuf)
    require.NoError(t, err)
    copiedArchive.Close()
    
    // Verify contents match
    // ...
}
```

### Files to modify
- Modify: `pkg/archive/zip/zip_test.go`
- Modify or Create: `pkg/archive/targz/targz_test.go`

### Difficulty
**Easy-Medium** - Requires creating archives and verifying their contents, but the pattern exists in other tests.
```

---

## Issue 4: Add tests for Target methods in internal/builders/poetry

**Title:** Add tests for Target.Fields() and Target.String() in internal/builders/poetry

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
The `internal/builders/poetry/build.go` file contains a `Target` type with two methods that have 0% test coverage.

### Affected Methods
- `Target.Fields()` (line 62)
- `Target.String()` (line 70)

### Current Coverage
**39.6%** → target **~50%** (also applies to internal/builders/uv which has the same pattern)

### What needs to be done
1. Add test cases in `internal/builders/poetry/build_test.go`
2. Test that `Target.Fields()` returns the expected map with "all" values
3. Test that `Target.String()` returns "none-any"

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
```

---

## Issue 5: Add tests for Env utility methods in pkg/context

**Title:** Add tests for Env.Copy() and Env.Strings() methods in pkg/context

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
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
    
    // Verify it's a separate copy
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
```

### Files to modify
- Modify: `pkg/context/context_test.go`

### Difficulty
**Very Easy** - Simple utility methods with straightforward testing requirements.
```

---

## Issue 6: Add tests for CheckSCM method in pkg/config

**Title:** Add test for Repo.CheckSCM() method in pkg/config

**Labels:** `good first issue`, `test coverage`, `enhancement`

**Body:**
```markdown
### Description
The `pkg/config/config.go` file contains a `CheckSCM()` method on the `Repo` type that has 0% test coverage.

### Affected Method
- `Repo.CheckSCM()` (line 65)

### Current Coverage
**46.1%** → target **~48%**

### What needs to be done
1. Add test cases in `pkg/config/config_test.go`
2. Test that valid SCM URLs are accepted (github.com, gitlab.com, gitea.io)
3. Test that invalid SCM URLs return an error
4. Test edge cases (empty URL, malformed URL, etc.)

### Example Test Structure
```go
func TestRepoCheckSCM(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr bool
    }{
        {"valid github", "https://github.com/owner/repo", false},
        {"valid gitlab", "https://gitlab.com/owner/repo", false},
        {"valid gitea", "https://gitea.io/owner/repo", false},
        {"invalid scm", "https://example.com/owner/repo", true},
        {"empty url", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Repo{SCMToken: SCMToken{URL: tt.url}}.CheckSCM()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Files to modify
- Modify: `pkg/config/config_test.go`

### Difficulty
**Easy** - Simple validation method that needs basic test cases.
```

---

## Summary

These 6 issues represent low-hanging fruit for test coverage improvements:

1. **JSONSchema methods** (8 methods, 0% coverage) - Very Easy
2. **NFPM conversion** (2 methods, 0% coverage) - Easy
3. **Archive Copy methods** (2 methods, 0% coverage) - Easy-Medium
4. **Poetry Target methods** (2 methods, 0% coverage) - Very Easy
5. **Env utility methods** (2 methods, 0% coverage) - Very Easy
6. **CheckSCM method** (1 method, 0% coverage) - Easy

**Total:** 17 uncovered methods that can be easily tested

**Expected overall impact:** These improvements would increase project coverage from ~80.7% to approximately 82-83%.

Each issue is tagged as `good first issue` because they're well-scoped, have clear acceptance criteria, and don't require deep knowledge of the codebase.
