---
title: "Add fuzzy testing for archive file processing"
labels: ["enhancement", "testing", "security", "files"]
---

## Description

Add fuzzy testing for the archive file processing module (`internal/archivefiles`) to improve security and robustness when handling file paths and glob patterns.

## Rationale

The archive files module:
- Handles file path processing and glob patterns
- Path traversal vulnerabilities are possible
- Glob pattern parsing is complex
- File destination calculation is critical for security
- Processes user-provided file paths and patterns

## Proposed Implementation

Create `internal/archivefiles/archivefiles_fuzz_test.go` with the following tests:

### 1. `FuzzEval`
Test file evaluation with various glob patterns:
- Malicious glob patterns
- Path traversal attempts (../, ../../, etc.)
- Very long file paths
- Unicode characters in paths
- Special characters (null bytes, control chars)
- Symbolic links
- Glob with invalid characters

### 2. `FuzzDestinationFor`
Test destination path calculation:
- Path traversal in destinations
- Conflicting strip_parent settings
- Very long destination paths
- Relative vs absolute paths
- Edge cases in filepath.Rel

### 3. `FuzzLongestCommonPrefix`
Test prefix calculation:
- Empty string arrays
- Single element arrays
- No common prefix
- Very long paths
- Unicode in paths

### 4. `FuzzMTimeFormatting`
Test modification time parsing:
- Invalid RFC3339 timestamps
- Very far past/future dates
- Timezone edge cases
- Malformed time strings

## Example Test Structure

```go
func FuzzEval(f *testing.F) {
    // Add seed corpus
    f.Add("**/*.go")
    f.Add("../../../etc/passwd")
    f.Add("*.{txt,md}")
    f.Add("/absolute/path/*")
    
    f.Fuzz(func(t *testing.T, pattern string) {
        ctx := testctx.New()
        tpl := tmpl.New(ctx)
        
        files := []config.File{
            {Source: pattern},
        }
        
        // Should not panic, may return error
        _, err := Eval(tpl, files)
        _ = err
    })
}

func FuzzDestinationFor(f *testing.F) {
    f.Add("dest/", "prefix/", "prefix/file.txt", false)
    f.Add("", ".", "../../../etc/passwd", false)
    f.Add("dest/", "src/", "src/very/deep/path/file.txt", true)
    
    f.Fuzz(func(t *testing.T, dest, prefix, path string, stripParent bool) {
        file := config.File{
            Destination: dest,
            StripParent: stripParent,
        }
        
        // Should not panic
        result, err := destinationFor(file, prefix, path)
        if err == nil {
            // If successful, result should not escape intended directory
            require.NotContains(t, result, "..")
        }
    })
}
```

## Acceptance Criteria

- [ ] Create `internal/archivefiles/archivefiles_fuzz_test.go`
- [ ] Implement at least 4 fuzz test functions
- [ ] Add security-focused seed corpus
- [ ] Test path traversal prevention
- [ ] Verify glob pattern handling is safe
- [ ] Test unicode and special character handling
- [ ] Integrate with existing test suite

## Related Files

- `internal/archivefiles/archivefiles.go`
- `internal/archivefiles/archivefiles_test.go`
- `pkg/config/config.go` (File struct)

## Priority

**High** - Archive file processing is security-critical and handles user input.
