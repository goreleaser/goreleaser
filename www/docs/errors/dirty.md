# Git is currently in a dirty state

GoReleaser requires a clean git state to work.

If you see this error, it means that something in your build process is either creating or editing files before GoReleaser is called.
The error message should show you which files were created/modified.

Here's an example error:

```sh
   ⨯ release failed after 0.02s error=git is currently in a dirty state
Please check in your pipeline what can be changing the following files:
 M modified.go
?? created.txt

Learn more at https://goreleaser.com/errors/dirty
```

From here on, you have a couple of options:

- add the file to `.gitignore` (recommended if the file is temporary and/or generated);
- change your process the build process to not touch any git tracked files.
