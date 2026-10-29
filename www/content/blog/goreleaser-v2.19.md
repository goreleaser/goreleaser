---
title: Announcing GoReleaser v2.19
date: 2026-10-22
slug: goreleaser-v2.19
tags: [announcements]
authors: [caarlos0]
---

A polish release: configurable Zstandard compression, declarative Homebrew cask
hooks, Nix publishing without Nix installed, and six builders that are no
longer experimental.

<!--more-->

## Zstandard compression settings

`tar.zst` and `tzst` archives can now set the compression level and the window
size:

```yaml {filename=".goreleaser.yaml"}
archives:
  - formats: [tar.zst]
    zstd:
      level: best
      window_size: 27
```

`level` accepts `fastest`, `default`, `better`, and `best`. `window_size` is
the log2 of the window size in bytes, from 10 to 29. A larger window can
improve compression when the archive holds binaries with shared code, at the
cost of more memory and time. If you omit the settings, nothing changes.

The settings also apply to meta archives and format overrides.

See the [documentation](https://goreleaser.com/customization/package/archives/#zstandard-compression)
for more details.

## Homebrew cask hooks with `install_steps`

Homebrew deprecated the `preflight`, `postflight`, `uninstall_preflight`, and
`uninstall_postflight` stanzas, so every generated cask that used a hook printed
a deprecation warning to your users.

The replacement stanzas don't accept arbitrary Ruby: they use Homebrew's
declarative install steps. GoReleaser can't translate a Ruby hook into them, so
there are new options instead:

```yaml {filename=".goreleaser.yaml"}
homebrew_casks:
  - name: myapp
    hooks:
      post:
        install_steps: |
          on_macos do
            run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{ .StagedPath }}/myapp"]
          end
```

`pre` and `post` both accept `install_steps` and `uninstall_steps`. The raw Ruby
`install` and `uninstall` options are deprecated; if you set both, the `_steps`
option wins.

See the [documentation](https://goreleaser.com/customization/publish/homebrew_casks/)
for more details.

## Nix without Nix

Publishing a Nix package used to require `nix-hash` in your `$PATH`, and the
whole pipe was skipped when it was missing, which is a surprising failure for a
package manager you may not otherwise use.

GoReleaser now computes the hash itself, so you don't need Nix installed to
publish to a Nix repository. There is nothing to configure.

See the [documentation](https://goreleaser.com/customization/publish/nix/) for
more details.

## Stable builders

The Zig, Rust, Bun, Deno, UV, and Poetry builders are no longer experimental,
and GoReleaser doesn't print the experimental warning for them anymore. The
`smartsemver` tag sort is stable too.

The MCP registry publisher, the Node.js SEA builder, and nFPM's `msix` stay
experimental for now.

See the [documentation](https://goreleaser.com/customization/builds/) for more
details.

## ko Linux capabilities

`ko` entries can now set Linux capabilities on the binary inside the image:

```yaml {filename=".goreleaser.yaml"}
kos:
  - repositories:
      - ghcr.io/myuser/myapp
    linux_capabilities:
      - net_bind_service
```

It accepts the same values as Docker's `--cap-add`. Setting it disables ko's
layer cache for that image.

Thanks to [Manuel Rüger](https://github.com/mrueg) for contributing this.

See the [documentation](https://goreleaser.com/customization/package/ko/) for
more details.

## Fixes since v2.18

Between v2.18 and now, the v2.18.1, v2.18.2, and v2.18.3 patch releases shipped
fixes worth knowing about:

- **security**: secrets are redacted from builder subprocess output
- [**blob**](https://goreleaser.com/customization/publish/blob/): unencrypted uploads are streamed instead of buffering whole files, thanks to [Ravi Yadav](https://github.com/raviayadav); KMS encryption limits and bucket lifetime on error are fixed
- [**docker**](https://goreleaser.com/customization/package/docker/): a bare platform means the baseline CPU variant, and arm64 minor variants stay above the v8 baseline
- [**builds**](https://goreleaser.com/customization/builds/): fixes for the Go, Rust, Zig, Node, Bun, UV, and Poetry builders, including canonical artifact architectures and target matching
- [**archive**](https://goreleaser.com/customization/package/archives/): stripped binary paths are reported correctly, thanks to [Davi Castello Branco Tavares de Oliveira](https://github.com/davicbtoliveira); ZIP metadata, mtime, and CPU variants are fixed
- [**aur**](https://goreleaser.com/customization/publish/aur/): several `PKGBUILD` and `.SRCINFO` generation bugs, and duplicate package names are now rejected

## Other updates

- [**archive**](https://goreleaser.com/customization/package/archives/): tar read errors are propagated when copying
- [**homebrew**](https://goreleaser.com/customization/publish/homebrew_casks/): emit valid casks when `builds.ignore` drops os/arch combinations; write completion stanzas in Homebrew's order
- [**docker**](https://goreleaser.com/customization/package/dockers_v2/): select baseline CPU variants for Docker v2 platforms; expand global `ARG` references when resolving the base image
- [**git**](https://goreleaser.com/customization/general/git/): redact credentials from HTTP remote URLs too
- [**ko**](https://goreleaser.com/customization/package/ko/): use the actual ko version in the SPDX SBOM creator
- [**upload**](https://goreleaser.com/customization/publish/upload/): share the HTTP client per upload config
- **security**: update to Go 1.27.3 and bump `golang.org/x/` dependencies
- **dependencies**: bump `moby/sys/user` to v0.4.1
- **pro/changelog**: AI changelog fixes — Ollama uses the OpenAI-compatible API, the tag field is the supported one, provider defaults are updated, and responses are validated and assembled
- **pro/schema**: fix a duplicate schema ID between `File` and `ExtraFile`

## Other news

- GoReleaser now has ~16.1k stars and 508 contributors! Thanks, everyone!
- You can follow release updates on our
  [Telegram channel](https://t.me/goreleasernews).
- nFPM had new releases as well,
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< g_button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.19.0" label="v2.19.0 (OSS)" icon="github" primary="false" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.19.0" label="v2.19.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< g_button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< g_button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}
