**Title:** Add tests for Copy methods in pkg/archive/zip and pkg/archive/targz

**Labels:** good first issue, test coverage, enhancement

**Description:**

Both `pkg/archive/zip/zip.go` and `pkg/archive/targz/targz.go` have `Copy` methods with 0% test coverage. There's even a TODO comment in `zip_test.go` line 169 that says "TODO: add copying test".

### Affected Methods
- `zip.Copy()` in `pkg/archive/zip/zip.go` (line 35)
- `targz.Copy()` in `pkg/archive/targz/targz.go` (line 30)

### Current Coverage
- `pkg/archive/zip`: **54.8%** → target **~75%**
- `pkg/archive/targz`: **46.2%** → target **~70%**

### What needs to be done
1. Add test for `zip.Copy()` in `pkg/archive/zip/zip_test.go`
2. Add test for `targz.Copy()` in a test file (create if needed)
3. Tests should:
   - Create a source archive with known contents
   - Copy it to a new archive
   - Verify the copied archive contains the same files
   - Verify file permissions and metadata are preserved

### Example Test Structure
```go
func TestZipCopy(t *testing.T) {
    tmp := t.TempDir()
    
    // Create source zip with test files
    sourcePath := filepath.Join(tmp, "source.zip")
    sourceFile, err := os.Create(sourcePath)
    require.NoError(t, err)
    
    sourceArchive := New(sourceFile)
    require.NoError(t, sourceArchive.Add(config.File{
        Source:      "../testdata/foo.txt",
        Destination: "foo.txt",
    }))
    require.NoError(t, sourceArchive.Close())
    require.NoError(t, sourceFile.Close())
    
    // Open source for reading
    sourceFile, err = os.Open(sourcePath)
    require.NoError(t, err)
    defer sourceFile.Close()
    
    // Copy to new archive
    targetPath := filepath.Join(tmp, "target.zip")
    targetFile, err := os.Create(targetPath)
    require.NoError(t, err)
    defer targetFile.Close()
    
    copiedArchive, err := Copy(sourceFile, targetFile)
    require.NoError(t, err)
    require.NoError(t, copiedArchive.Close())
    require.NoError(t, targetFile.Close())
    
    // Verify target archive has same contents
    targetFile, err = os.Open(targetPath)
    require.NoError(t, err)
    defer targetFile.Close()
    
    info, err := targetFile.Stat()
    require.NoError(t, err)
    
    r, err := zip.NewReader(targetFile, info.Size())
    require.NoError(t, err)
    require.Len(t, r.File, 1)
    require.Equal(t, "foo.txt", r.File[0].Name)
}
```

### Files to modify
- Modify: `pkg/archive/zip/zip_test.go`
- Create or Modify: `pkg/archive/targz/targz_test.go`

### Difficulty
**Easy-Medium** - Requires creating archives and verifying their contents, but the pattern exists in other tests.
