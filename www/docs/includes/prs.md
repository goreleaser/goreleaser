## Pull Requests

> Since v1.17

GoReleaser allows you to, instead of pushing directly to the main branch, push
to a feature branch, and open a pull requests with the changes.

### Templates

> Since v1.19

GoReleaser will check for a `.github/PULL_REQUEST_TEMPLATE.md`, and set it in
the pull request body if it exists.

We do that to prevent extra work for maintainers of things like `winget-pkgs`,
`nixpkgs`, and so on.

### Cross-repository pull requests

> Since v1.19

You can also push to a fork, and open the pull request in the original branch.

Here's an example on how to set it up:

```yaml
# .goreleaser.yml
# ...
something: # can be nix, brews, etc...
  - repository:
      owner: john
      name: repo
      branch: "{{.ProjectName}}-{{.Version}}"
      pull_request:
        enabled: true
        base:
          owner: mike
          name: repo
          branch: main
```

This will:

- Create the files into `john/repo`, in the branch `foo-1.2.3` (assuming
  `ProjectName=foo` and `Version=1.2.3`). [^head]
- Open a pull request from `john/repo` into `mike/repo`, with the branch `main`
  as target. [^base]

[^head]: In GitHub's terms, this means `head=john:repo:foo-1.2.3`
[^base]: In GitHub's terms, this means `base=mike:repo:main`

### Things that don't work

- **GoReleaser will not keep your fork in sync!!!** It might or might not be a
  problem in your case, in which case you'll have to sync it manually.
- Opening pull requests to a forked repository (`go-github` does not have the
  required fields to do it).
- Since this can fail for a myriad of reasons, if an error happen, it'll log it
  to the release output, but will not fail the pipeline.
