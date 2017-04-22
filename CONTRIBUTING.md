# Contributing

By participating to this project, you agree to abide our [code of
conduct](/CODE_OF_CONDUCT.md).

## Setup your machine

`goreleaser` is written in [Go](https://golang.org/).

Prerequisites are:

* Build:
  * `make`
  * [Go 1.8+](http://golang.org/doc/install)

Clone `goreleaser` from source into `$GOPATH`:

```sh
$ go get -d github.com/goreleaser/goreleaser
$ cd $GOPATH/src/github.com/goreleaser/goreleaser
```

Install the build and lint dependencies:

``` sh
$ make setup
```

A good way of making sure everything is all right is running the test suite:

``` sh
$ make test
```

## Test your change

You can create a branch for your changes and try to build from the source as you go:

``` sh
$ make build
```

When you are satisfied with the changes, we suggest you run:

``` sh
$ make ci
```

Which runs all the linters and tests.

## Submit a pull request

Push your branch to your `goreleaser` fork and open a pull request against the
master branch.
