---
title: "Add fuzzy testing for Cargo.toml parsing"
labels: ["enhancement", "testing", "rust"]
---

## Description

Add fuzzy testing for the Cargo.toml parsing module (`internal/cargo`) to improve robustness when processing Rust Cargo manifest files.

## Rationale

The Cargo.toml parser:
- Parses Rust project manifest files
- Workspace member parsing is critical for monorepos
- TOML parsing vulnerabilities could affect builds
- Handles external input from Rust projects

## Proposed Implementation

Create `internal/cargo/cargo_fuzz_test.go` with the following tests:

### 1. `FuzzOpen`
Test Cargo.toml file parsing:
- Malformed TOML structures
- Invalid package names
- Malformed workspace configurations
- Very large member lists
- Missing sections
- Invalid field types

### 2. `FuzzWorkspaceMembers`
Test workspace member parsing:
- Empty member lists
- Duplicate members
- Very long member paths
- Special characters in paths
- Glob patterns in members

## Example Test Structure

```go
func FuzzOpen(f *testing.F) {
    f.Add([]byte(`[package]
name = "mypackage"
version = "0.1.0"`))
    f.Add([]byte(`[workspace]
members = ["crate1", "crate2"]`))
    f.Add([]byte(`invalid toml`))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        tmpfile := filepath.Join(t.TempDir(), "Cargo.toml")
        require.NoError(t, os.WriteFile(tmpfile, data, 0o644))
        
        cargo, err := Open(tmpfile)
        if err != nil {
            return // Expected for invalid TOML
        }
        
        // Access fields - should not panic
        _ = cargo.Package.Name
        _ = cargo.Workspace.Members
    })
}
```

## Acceptance Criteria

- [ ] Create `internal/cargo/cargo_fuzz_test.go`
- [ ] Implement at least 2 fuzz test functions
- [ ] Add seed corpus with valid and invalid Cargo.toml examples
- [ ] Tests handle malformed TOML gracefully
- [ ] Verify workspace member parsing handles edge cases
- [ ] Integrate with existing test suite

## Related Files

- `internal/cargo/cargo.go`
- `internal/cargo/cargo_test.go`
- `internal/cargo/testdata/`

## Priority

**High** - Cargo.toml parsing is critical for Rust project support.
