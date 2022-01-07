# Introduction

GoReleaser is a release automation tool for Go projects.
The goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

GoReleaser is built with CI tools in mind, you only need to download and execute it in your build script.
Of course, you can still install it locally if you want.

Your release process can be customized through a `.goreleaser.yaml` file.

Once you set it up, every time you want to create a new release, all you need to do is tag and run `goreleaser release`:

<script id="asciicast-385826" src="https://asciinema.org/a/385826.js" async></script>
