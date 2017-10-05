# Contributing

By participating to this project, you agree to abide our [code of
conduct](/CODE_OF_CONDUCT.md).


## Setup your machine

`goreleaser` is written in [Go](https://golang.org/).

Prerequisites:

* `make`
* [Go 1.8+](https://golang.org/doc/install)
* [fpm](https://fpm.readthedocs.io/en/latest/installing.html)
* rpm / rpmbuild
* [snapcraft](https://snapcraft.io/)
* [Docker](https://www.docker.com/)

Clone `goreleaser` from source into `$GOPATH`:

```sh
$ mkdir -p $GOPATH/src/github.com/goreleaser/goreleaser
$ cd $_
$ git clone git@github.com:goreleaser/goreleaser.git
$ cd goreleaser
```

Install the build and lint dependencies:

```console
$ make setup
```

A good way of making sure everything is all right is running the test suite:

```console
$ make test
```


## Test your change

You can create a branch for your changes and try to build from the source as you go:

```console
$ make build
```

When you are satisfied with the changes, we suggest you run:

```console
$ make ci
```

Which runs all the linters and tests.


## Create a commit

Commit messages should be well formatted.
Start your commit message with the type. Choose one of the following:
`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `revert`, `add`, `remove`, `move`, `bump`, `update`, `release`

After a colon, you should give the message a title, starting with uppercase and ending without a dot.
Keep the width of the text at 72 chars.
The title must be followed with a newline, then a more detailed description.

Please reference any GitHub issues on the last line of the commit message (e.g. `See #123`, `Closes #123`, `Fixes #123`).

An example:

```
docs: Add example for --release-notes flag

I added an example to the docs of the `--release-notes` flag to make
the usage more clear.  The example is an realistic use case and might
help others to generate their own changelog.

See #284
```


## Submit a pull request

Push your branch to your `goreleaser` fork and open a pull request against the
master branch.
