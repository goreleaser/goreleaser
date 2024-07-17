# Contributing

By participating in this project, you agree to abide our
[code of conduct](https://github.com/goreleaser/.github/blob/main/CODE_OF_CONDUCT.md).

## Set up your machine

`goreleaser` is written in [Go](https://go.dev/).

Prerequisites:

- [Task](https://taskfile.dev/installation)
- [Go 1.22+](https://go.dev/doc/install)

Other things you might need to run the tests:

- [cosign](https://github.com/sigstore/cosign)
- [Docker](https://www.docker.com/)
- [GPG](https://gnupg.org)
- [Podman](https://podman.io/)
- [Snapcraft](https://snapcraft.io/)
- [Syft](https://github.com/anchore/syft)
- [upx](https://upx.github.io/)

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

### A note about Docker multi-arch builds

If you want to properly run the Docker tests, or run `goreleaser release
--snapshot` locally, you might need to setup Docker for it.
You can do so by running:

```sh
task docker:setup
```

## Test your change

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

## Create a commit

Commit messages should be well formatted, and to make that "standardized", we
are using Conventional Commits.

You can follow the documentation on
[their website](https://www.conventionalcommits.org).

## Submit a pull request

Push your branch to your `goreleaser` fork and open a pull request against the main branch.

## Financial contributions

You can contribute in our OpenCollective or to any of the contributors directly.
See [this page](https://goreleaser.com/sponsors) for more details.
