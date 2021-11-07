# Contributing

By participating to this project, you agree to abide our [code of conduct](https://github.com/goreleaser/goreleaser/blob/master/CODE_OF_CONDUCT.md).

## Setup your machine

`goreleaser` is written in [Go](https://golang.org/).

Prerequisites:

- [Task](https://taskfile.dev/#/installation)
- [Go 1.17+](https://golang.org/doc/install)

Other things you might need to run the tests:

- [Buildpacks](https://buildpacks.io/)
- [cosign](https://github.com/sigstore/cosign)
- [Docker](https://www.docker.com/)
- [GPG](https://gnupg.org)
- [Podman](https://podman.io/)
- [Snapcraft](https://snapcraft.io/)

Clone `goreleaser` anywhere:

```sh
git clone git@github.com:goreleaser/goreleaser.git
```

`cd` into the directory and install the dependencies:

```sh
task setup
```

A good way of making sure everything is all right is running the test suite:

```sh
task test
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

## Create a commit

Commit messages should be well formatted, and to make that "standardized", we
are using Conventional Commits.

You can follow the documentation on
[their website](https://www.conventionalcommits.org).

## Submit a pull request

Push your branch to your `goreleaser` fork and open a pull request against the
master branch.

## Financial contributions

We also welcome financial contributions in full transparency on our [open collective](https://opencollective.com/goreleaser).
Anyone can file an expense. If the expense makes sense for the development of the community, it will be "merged" in the ledger of our open collective by the core contributors and the person who filed the expense will be reimbursed.
