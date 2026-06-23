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
{{< card link="go" title="Golang" icon="go" >}}
{{< card link="rust" title="Rust" icon="rust" >}}
{{< card link="node" title="Node.js" tag="new" icon="node" >}}
{{< card link="zig" title="Zig" icon="zig" >}}
{{< card link="bun" title="Bun" icon="bun" >}}
{{< card link="deno" title="Deno" icon="deno" >}}
{{< card link="uv" title="UV" icon="uv" >}}
{{< card link="poetry" title="Poetry" icon="poetry" >}}
{{< card link="python" title="Python" tag="soon" icon="python" >}}
{{< card link="prebuilt" title="Import from other build systems" icon="variable" >}}
{{< /cards >}}
