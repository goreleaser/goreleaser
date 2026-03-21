---
title: "Build"
weight: 20
---

GoReleaser supports multiple programming languages and build
systems through its _builder_ interfaces.

A _builder_ gets a build configuration and emits binaries/libraries into the
`dist` directory.

## Supported builders

{{< cards cols="2" >}}
{{< card link="builders/go" title="Golang" >}}
{{< card link="builders/rust" title="Rust" >}}
{{< card link="builders/zig" title="Zig" >}}
{{< card link="builders/bun" title="Bun" >}}
{{< card link="builders/deno" title="Deno" >}}
{{< card link="builders/uv" title="UV" >}}
{{< card link="builders/poetry" title="Poetry" >}}
{{< card link="builders/python" title="Python" tag="soon">}}
{{< card link="builders/prebuilt" title="Import from other build systems" >}}
{{< /cards >}}
