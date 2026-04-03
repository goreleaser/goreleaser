# Contributing

By participating in this project, you agree to abide our
[code of conduct](https://github.com/goreleaser/.github/blob/main/CODE_OF_CONDUCT.md).

## Set up your machine

`goreleaser` is written in [Go](https://go.dev/).

Prerequisites:

- [Task](https://taskfile.dev/installation)
- [Go 1.26+](https://go.dev/doc/install)

Other things you might need to run some of the tests (they should get
automatically skipped if a needed tool isn't present):

- [cosign](https://github.com/sigstore/cosign)
- [Docker](https://www.docker.com/)
- [GPG](https://gnupg.org)
- [Podman](https://podman.io/)
- [Snapcraft](https://snapcraft.io/)
- [Syft](https://github.com/anchore/syft)
- [upx](https://upx.github.io/)

## Building

Clone `goreleaser` anywhere:

```sh
git clone git@github.com:goreleaser/goreleaser.git
```

`cd` into the directory and install the dependencies:

```bash
go mod tidy
```

You should then be able to build the binary:

```bash
go build -o goreleaser .
./goreleaser --version
```

## Testing your changes

You can create a branch for your changes and try to build from the source as you go:

```sh
task build
```

When you are satisfied with the changes, we suggest you run:

```sh
task ci
```

Before you commit the changes, we also suggest you run:

```sh
task fmt
```

### A note about Docker multi-arch builds

If you want to properly run the Docker tests, or run `goreleaser release
--snapshot` locally, you might need to setup Docker for it.
You can do so by running:

```sh
task docker:setup
```

### A note about Windows

Make sure to enable "Developer Mode" in Settings.

## Writing pipes

Pipes should follow these conventions for consistent error output:

- **Do not prefix error messages with the pipe name.** Error wrapping with
  the pipe name is done at the meta-pipe level (`publish`, `announce`,
  `defaults`), so individual pipes should only describe the problem itself
  (e.g., `"no archives found"` instead of `"archive: no archives found"`).
- **Use the pipe's `String()` method for context.** Meta-pipes wrap errors
  using `fmt.Errorf("%s: %w", pipe.String(), err)`, which provides
  consistent, non-redundant context.

## Creating a commit

Commit messages should be well formatted, and to make that "standardized", we
are using Conventional Commits.

You can follow the documentation on
[their website](https://www.conventionalcommits.org).

## AI usage guidelines

AI usage is permitted as long as all these rules are followed:

1. It is disclosed, either with a `Co-authored-by` or `Assisted-by` in the
   commit message, or if you specify in the PR description
2. You have to disclose to what extend AI was used
3. You (as an human) have to review all the work the AI has done

Agents that keep going on random repositories pretending to be a human doing
things are not allowed either.

We reserve the right to close any and all issues/PRs/discussions that do not
follow these rules.

## Submitting a pull request

Push your branch to your `goreleaser` fork and open a pull request against the main branch.

## Financial contributions

You can contribute in our OpenCollective or to any of the contributors directly.
See [this page](https://goreleaser.com/sponsors) for more details.
