1. Extract the hashing logic into a separate function `generateManifestLine(recordType, path, manifestHashes []string, content []byte) (string, error)` or similar.
2. But wait, `content` logic and hashing logic is duplicated between DIST lines (lines 1157-1209) and the new thick manifest lines (lines 1211-1281).
3. Let's create a helper `generateManifestLine(recordType, filename string, size int64, f io.Reader, manifestHashes []string) (string, error)` to handle both? Or just helper `hashContent(content []byte, hashes []string)`... wait, DIST is reading from file directly.

Let's look at DIST generation:
```go
		err := func() error {
			info, err := os.Stat(art.Path)
			if err != nil {
				return err
			}
			size := info.Size()

			f, err := os.Open(art.Path)
			if err != nil {
				return err
			}
			defer f.Close()

			var writers []io.Writer
			var b2b hash.Hash
            // ...
```
And new thick manifest:
```go
			err := func() error {
				content := f.Content
				if content == nil && f.Path != "" {
					var err error
					content, err = os.ReadFile(f.Path)
					if err != nil {
						return err
					}
				}
				size := int64(len(content))

				var b2b hash.Hash
                // ...
```

Let's refactor into a single helper function:
```go
func appendManifestRecord(manifestLines *[]string, recordType, filename string, size int64, r io.Reader, manifestHashes []string) error {
    // ... setup writers ...
    // io.Copy
    // format string
    // append to manifestLines
    return nil
}
```

Wait, `appendManifestRecord` would be very clean.
Let's see if we can do this.
