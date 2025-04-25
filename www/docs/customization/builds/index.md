# Introduction

GoReleaser supports multiple programming languages and build
systems through its _builder_ interfaces.

A _builder_ gets a build configuration and emits binaries/libraries into the
`dist` directory.

Multiple builders are supported, with more coming soon.

Here's the list:

<div class="grid cards" markdown>

- :simple-go: [Golang](./go.md)
- :simple-rust: [Rust](./rust.md)[^v2.5]
- :simple-zig: [Zig](./zig.md)[^v2.5]
- :simple-bun: [Bun](./bun.md)[^v2.6]
- :simple-deno: [Deno](./deno.md)[^v2.6]
- :simple-uv: [UV](./uv.md)[^v2.9]
- :simple-poetry: [Poetry](./poetry.md)[^v2.9]
- :simple-python: [Python/Pip](./python.md)[^soon]
- :material-asterisk: [Import from other build systems](../prebuilt.md)

</div>

[^v2.5]: Added in v2.5.

[^v2.6]: Added in v2.6.

[^v2.9]: Added in v2.9 (unreleased).

[^soon]: Coming in a future version.
