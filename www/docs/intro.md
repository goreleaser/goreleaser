# Introduction

Putting it simply, GoReleaser is a release automation tool for Go projects.

## Why we made it?

GoReleaser was created to solve a problem we all had at some point: releasing
software is boring and error prone.

To fix that, we all end up creating scripts to automate the work, with various
levels of success.

Generally speaking, those scripts tend to not be reusable and have dependencies
on many other tools - which makes it hard to run the process on other machines.

GoReleaser aims to make all these scripts obsolete: instead of writing scripts,
you write a simple YAML configuration file; instead of many tools, you (usually)
only need a single `goreleaser` binary.

Then, you can simply run a single command to build, archive, package, sign and
publish artifacts.

We work hard to make it easy for you, our user, to do the best thing for your
users.
That's why we focus on providing easy-to-use integrations, good defaults and
many tutorials with tools that help mitigate supply chain security issues,
package managers, go mod proxying and so on.

This way its easy to provide easy to install packages, with signed checksums,
software bill of materials, and reproducible binaries, for example.

## Is it any good?

GoReleaser has been widely adopted by the Go community in the past few years,
with
[thousands of projects and companies](https://github.com/search?q=path%3A.goreleaser.yml+OR+path%3A.goreleaser.yaml+&type=code)
using it to manage their releases.

You can check some of our users out [here](users.md).

## Use cases

GoReleaser is built with CI tools in mind — you only really need to download and
execute it in your build script.

Installing it in your machine is entirely up to you, but still possible.

## Usage

Your entire release process is customized through a `.goreleaser.yaml` file.

Once you set it up, every time you want to create a new release, all you need to
do is push a git tag and run `goreleaser release`:

![goreleaser example gif](https://raw.githubusercontent.com/goreleaser/example-simple/main/goreleaser.gif)

You can also do it in your continuous integration platform of choice.

---

Hopefully you find it useful, and the docs easy to follow.

Feel free to [create an issue][iss] if you find something that's not clear and
join our [Discord][dis] to chat with other users and maintainers.

[iss]: https://github.com/goreleaser/goreleaser/issues
[dis]: https://discord.gg/RGEBtg8vQ6
