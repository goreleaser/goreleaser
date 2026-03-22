---
title: "Builders"
weight: 1
---

GoReleaser supports multiple programming languages and build
systems through its _builder_ interfaces.

A _builder_ gets a build configuration and emits binaries/libraries into the
`dist` directory.

## Supported builders

{{< cards cols="2" >}}
{{< card link="go" title="Golang" >}}
{{< card link="rust" title="Rust" >}}
{{< card link="zig" title="Zig" >}}
{{< card link="bun" title="Bun" >}}
{{< card link="deno" title="Deno" >}}
{{< card link="uv" title="UV" >}}
{{< card link="poetry" title="Poetry" >}}
{{< card link="python" title="Python" tag="soon">}}
{{< card link="prebuilt" title="Import from other build systems" >}}
{{< /cards >}}
