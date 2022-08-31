# Introduction

GoReleaser is a release automation tool for Go projects.
The goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

It has been widely adopted by the Go community in the past years, with [thousands of projects](https://github.com/search?l=&q=filename%3Agoreleaser+language%3Ayaml+-path%3A%2Fvendor&type=code) using it to manage their releases.
You can check some of our users out [here](/users).

GoReleaser was built with CI tools in mind — you only really need to download and execute it in your build script.
Installing it in your machine is optional.

Your entire release process can be customized through a `.goreleaser.yml` file.
Once you set it up, every time you want to create a new release, all you need to do is tag and run `goreleaser release`:

<script id="asciicast-385826" src="https://asciinema.org/a/385826.js" async></script>

Hopefully you find it useful, and the docs easy to follow.
Feel free to [create an issue][iss] if you find something that's not clear and join our [Discord][dis] to chat with other users and maintainers.

[iss]: https://github.com/goreleaser/goreleaser/issues
[dis]: https://discord.gg/RGEBtg8vQ6
