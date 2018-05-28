# Contributing

By participating to this project, you agree to abide our [code of
conduct](/CODE_OF_CONDUCT.md).

## Setup your machine

`goreleaser` is written in [Go](https://golang.org/).

Prerequisites:

- `make`
- [Go 1.10+](https://golang.org/doc/install)
- `rpmbuild` (`apt get install rpm`/`brew install rpm`)
- [snapcraft](https://snapcraft.io/)
- [Docker](https://www.docker.com/)

Clone `goreleaser` from source into `$GOPATH`:

```sh
$ go get -d github.com/goreleaser/goreleaser
$ cd $GOPATH/src/github.com/goreleaser/goreleaser
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

## Financial contributions

We also welcome financial contributions in full transparency on our [open collective](https://opencollective.com/goreleaser).
Anyone can file an expense. If the expense makes sense for the development of the community, it will be "merged" in the ledger of our open collective by the core contributors and the person who filed the expense will be reimbursed.

## Credits

### Contributors

Thank you to all the people who have already contributed to goreleaser!
<a href="graphs/contributors"><img src="https://opencollective.com/goreleaser/contributors.svg?width=890" /></a>

### Backers

Thank you to all our backers! [[Become a backer](https://opencollective.com/goreleaser#backer)]

<a href="https://opencollective.com/goreleaser#backers" target="_blank"><img src="https://opencollective.com/goreleaser/backers.svg?width=890"></a>

### Sponsors

Thank you to all our sponsors! (please ask your company to also support this open source project by [becoming a sponsor](https://opencollective.com/goreleaser#sponsor))

<a href="https://opencollective.com/goreleaser/sponsor/0/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/1/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/2/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/3/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/4/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/5/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/6/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/7/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/8/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/goreleaser/sponsor/9/website" target="_blank"><img src="https://opencollective.com/goreleaser/sponsor/9/avatar.svg"></a>
