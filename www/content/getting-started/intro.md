---
title: "Introduction"
weight: 10
---

GoReleaser is a release automation tool.

It currently supports Go, Rust, Zig, JavaScript and TypeScript (with Node.js,
Bun, and Deno), and Python.

## Why we made it?

GoReleaser was created to solve a problem we all had at some point: releasing
software is boring and error prone.

To fix that, we all end up creating scripts to automate the work, with various
levels of success.

Generally speaking, those scripts tend to not be reusable and have dependencies
on many other tools — which makes it hard to run the process on other machines.

GoReleaser aims to make all these scripts obsolete: instead of writing scripts,
you write a simple YAML configuration file; instead of many tools, you (usually)
only need a single `goreleaser` binary.

Then, you can run a single command to build, archive, package, sign, and publish
artifacts.

We work hard to make it easy for you, our user, to do the best thing for your
users.
That's why we focus on easy-to-use integrations and good defaults for the tools
that matter: supply chain security, package managers, Go module proxying, and so
on.

That makes it easy to ship packages your users can install in one step, with
signed checksums, a software bill of materials, and reproducible binaries.

## Is it any good?

GoReleaser has been widely adopted by the Go community, with
[thousands of projects and companies](https://github.com/search?q=path%3A.goreleaser.yml+OR+path%3A.goreleaser.yaml+&type=code)
using it to manage their releases.

Browse the [list of users](/resources/users/) to see some of them.

## Use cases

GoReleaser is built with CI tools in mind — you only really need to download and
execute it in your build script.

You can also install it on your own machine, but you don't have to.

## Usage

Your entire release process is customized through a `.goreleaser.yaml` file.

Once you set it up, every time you want to create a new release, all you need to
do is push a git tag and run `goreleaser release`:

![Terminal recording of a GoReleaser run building, archiving, and publishing a release](https://raw.githubusercontent.com/goreleaser/example-simple/main/goreleaser.gif)

You can also do it in your continuous integration platform of choice.

---

Hopefully you find it useful, and the docs easy to follow.

Feel free to [create an issue][iss] if you find something that's not clear, and
to join [GitHub Discussions][dis] to chat with other users and maintainers.

[iss]: https://github.com/goreleaser/goreleaser/issues
[dis]: https://github.com/goreleaser/goreleaser/discussions
