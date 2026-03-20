---
title: "Introduction"
aliases:
  - "/customization/build/"
weight: 20
---

# Introduction

GoReleaser supports multiple programming languages and build
systems through its _builder_ interfaces.

A _builder_ gets a build configuration and emits binaries/libraries into the
`dist` directory.

Multiple builders are supported, with more coming soon.

Here's the list:

<div class="grid cards" markdown>

- :simple-go: [Golang](./go/)
- :simple-rust: [Rust](./rust/)
- :simple-zig: [Zig](./zig/)
- :simple-bun: [Bun](./bun/)
- :simple-deno: [Deno](./deno/)
- :simple-uv: [UV](./uv/)[^v2.9]
- :simple-poetry: [Poetry](./poetry/)[^v2.9]
- :simple-python: [Python/Pip](./python/)[^soon]
- :material-asterisk: [Import from other build systems](../prebuilt/)

</div>

[^v2.9]: Added in v2.9.

[^soon]: Coming in a future version.
