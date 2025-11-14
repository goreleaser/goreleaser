# Manual Issue Creation Guide

If automated scripts don't work, use this guide to manually create each issue by copy-pasting the content.

## Issue 1: YAML Configuration Parsing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for YAML configuration parsing
```

**Labels**: `enhancement`, `testing`, `security`

**Body**: Copy everything below the line
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

---

## Issue 2: Package.json Parsing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for package.json parsing
```

**Labels**: `enhancement`, `testing`, `nodejs`, `bun`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/02-packagejson-parsing.md` (skip YAML front matter)

---

## Issue 3: Pyproject.toml Parsing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for pyproject.toml parsing
```

**Labels**: `enhancement`, `testing`, `python`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/03-pyproject-parsing.md` (skip YAML front matter)

---

## Issue 4: Cargo.toml Parsing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for Cargo.toml parsing
```

**Labels**: `enhancement`, `testing`, `rust`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/04-cargo-parsing.md` (skip YAML front matter)

---

## Issue 5: Template Engine Expansion

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Expand fuzzy testing coverage for template engine
```

**Labels**: `enhancement`, `testing`, `templates`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/05-template-engine.md` (skip YAML front matter)

---

## Issue 6: Config Loading

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for config loading
```

**Labels**: `enhancement`, `testing`, `config`, `security`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/06-config-loading.md` (skip YAML front matter)

---

## Issue 7: Archive Files Processing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for archive file processing
```

**Labels**: `enhancement`, `testing`, `security`, `files`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/07-archive-files.md` (skip YAML front matter)

---

## Issue 8: Shell Command Construction

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for shell command construction
```

**Labels**: `enhancement`, `testing`, `security`, `critical`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/08-shell-commands.md` (skip YAML front matter)

---

## Issue 9: Changelog Parsing

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for changelog parsing
```

**Labels**: `enhancement`, `testing`, `changelog`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/09-changelog-parsing.md` (skip YAML front matter)

---

## Issue 10: HTTP Client Utilities

**URL**: https://github.com/goreleaser/goreleaser/issues/new

**Title**:
```
Add fuzzy testing for HTTP client utilities
```

**Labels**: `enhancement`, `testing`, `security`, `http`

**Body**: See `.github/ISSUE_TEMPLATES_FUZZING/10-http-handling.md` (skip YAML front matter)

---

## Tips for Manual Creation

1. Open each template file in `.github/ISSUE_TEMPLATES_FUZZING/`
2. Skip the YAML front matter (lines between `---`)
3. Copy everything after the second `---`
4. Paste into the issue body
5. Add labels as specified
6. Submit

## Faster Approach

Use the provided scripts instead:
- Python: `python3 scripts/create-fuzzing-issues.py`
- Bash: `./scripts/create-fuzzing-issues.sh`

Both require proper GitHub authentication but will create all 10 issues automatically.
