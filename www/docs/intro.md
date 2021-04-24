# Introduction

GoReleaser is a release automation tool for Go projects.
The goal is to simplify the build, release and publish steps while providing
variant customization options for all steps.

GoReleaser is built for CI tools, you only need to download and execute it in
your build script. Of course, you can also install it locally if you wish.

You can also customize your release process through a `.goreleaser.yml` file.

Once you set it up, every time you want to create a new release, all you need to do is tag and run
`goreleaser release`:

<script id="asciicast-385826" src="https://asciinema.org/a/385826.js" async></script>
