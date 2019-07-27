---
title: Introduction
weight: 1
menu: true
---

[GoReleaser](https://github.com/goreleaser/goreleaser) is a release automation
tool for Go projects. The goal is to simplify the build, release and
publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to
[download and execute it](/ci) in your build script.
Of course, you can also [install it locally](/install) if you wish.

You can also [customize](/customization) your release process through a
`.goreleaser.yml` file.

<span id="count" title="value get with goreleaser/func">Several</span>
GitHub projects trust their release process to GoReleaser.

<script>
var req = new XMLHttpRequest();
req.open("GET", "https://func.goreleaser.now.sh");
req.onload = function() {
  document.querySelector("#count").textContent = req.response
}
req.send();
</script>

<style>
  #count {
    font-weight: bold;
  }
</style>
