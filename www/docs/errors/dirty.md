# Git is in a dirty state

GoReleaser requires a clean git state to work.

If you see this error, it means that something in your build process is either
creating or editing files before GoReleaser is called. The error message should
show you, which files were created/modified.

Here's an example error:

```
   ⨯ release failed after 0.02s error=git is currently in a dirty state
Please check in your pipeline what can be changing the following files:
 M modified.go
?? created.txt

Learn more at https://goreleaser.com/errors/dirty
```

From here on, you have a couple of options:

- add the file to `.gitignore` (recommended if the file is temporary and/or
  generated);
- change your build process to not touch any git tracked files.
- if you are running `goreleaser build`, you might want to add either the
  `--snapshot` or `--skip=validate` flags to it

!!! tip "./dist"

    The `dist` folder (usually `./dist`) needs to be added to `.gitignore`, or
    deleted before running GoReleaser.
    `goreleaser init` takes care of that for you, if you used it to start your
    project, if not, you'll need to do it manually:

    ```sh
    echo './dist' >>.gitignore
    ```
