---
title: Node.js
weight: 25
---

<!-- markdownlint-disable MD025 -->

{{< experimental >}}

You can build Node.js [Single Executable Application][sea] (SEA) binaries
with GoReleaser, in pure Go — no `postject` or any other npm tooling
required. GoReleaser shells out to a host-platform Node.js (≥ v25.5.0,
auto-downloaded) once per build to invoke `node --build-sea`, and
downloads the per-target Node.js binary that becomes the SEA executable.

[sea]: https://nodejs.org/api/single-executable-applications.html

## Quick start

For a brand-new project:

```sh
goreleaser init --language node
```

This drops a starter `.goreleaser.yaml` next to your `package.json`.
For an existing project, copy the snippet under
[Configuration](#configuration) into your config and adjust `binary`,
`main`, and `targets`.

## How it works

For each requested target, GoReleaser:

1. Resolves the build-tool Node.js used to drive `--build-sea` (see
   [Build-tool Node.js](#build-tool-nodejs)).
2. Resolves the target Node.js version (see
   [Version resolution](#version-resolution)).
3. Downloads the official Node.js binary for that target from
   <https://nodejs.org/dist>, verifying its SHA-256 against the matching
   `SHASUMS256.txt` entry, and caches it under
   `${XDG_CACHE_HOME:-$HOME/.cache}/goreleaser/node/` so subsequent
   builds are offline.
4. Writes a `sea-config.json` in a scratch directory pointing `main`
   at the entrypoint script and `executable` at the cached target Node
   binary.
5. Invokes `<build-tool-node> --build-sea sea-config.json`. Node.js
   (LIEF-backed since v25.5) injects the SEA blob into a copy of the
   target Node binary and writes the result into the scratch directory.
6. On macOS targets only: applies an ad-hoc signature with
   `codesign --sign - --force` so the kernel loader will accept the
   binary. If `codesign(1)` is not on `PATH` (typical when
   cross-compiling for macOS from a non-darwin host) the binary is
   left unsigned and the [`signs`](/customization/sign/) pipe must
   re-sign it on a darwin runner before distribution.
7. Renames the result into the configured output path.

## Bundling your app

GoReleaser does **not** bundle your `node_modules/` for you, but it
does run `npm run build` automatically when your `package.json`
declares a `scripts.build` entry — so the file referenced by
[`main`](#configuration) is the freshly bundled output.

```json {filename="package.json"}
{
  "scripts": {
    "build": "esbuild src/index.js --bundle --platform=node --outfile=dist/bundle.js"
  }
}
```

```yaml {filename=".goreleaser.yaml"}
builds:
  - builder: node
    main: dist/bundle.js
```

That's it — no GoReleaser config needed. When `scripts.build` is
absent, the step is skipped silently and GoReleaser uses
[`main`](#configuration) as-is.

Dependency installation (`npm ci`, `pnpm install --frozen-lockfile`,
…) is **not** run for you — drive it from the global
[`before`](/customization/global_hooks/) hook so it executes once per
release rather than once per build:

```yaml {filename=".goreleaser.yaml"}
before:
  hooks:
    - npm ci

builds:
  - builder: node
    main: dist/bundle.js
```

If you need per-target bundling (e.g. different output for darwin
vs. linux), bypass the auto-step by giving your `package.json` no
`scripts.build` entry and drive the bundling from a per-target
[`hooks.pre`](/customization/build/) instead.


## Configuration

```yaml {filename=".goreleaser.yaml"}
builds:
  - id: my-build
    builder: node

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # Entrypoint script bundled into the SEA blob.
    #
    # Default: 'index.js'.
    # Templates: allowed.
    main: index.js

    # Pin the target Node.js version. Either an exact version
    # (`v22.20.0`, `22.20.0`) or a semver range (`>=22.20 <23`,
    # `^24.6`). Recommended for reproducible release artifacts.
    # When unset, falls back to engines.node / .nvmrc / .node-version.
    #
    # Templates: allowed.
    node_version: "22.20.0"

    # Targets, in nodejs.org/dist format.
    # Default: all of: darwin-arm64, darwin-x64, linux-arm64, linux-x64,
    #                  win-arm64, win-x64.
    # Other published targets that you can list explicitly:
    #   aix-ppc64, linux-armv7l, linux-ppc64le, linux-s390x.
    targets:
      - linux-x64
      - darwin-arm64

    # Path to the project's (sub)directory containing the code and
    # (typically) package.json.
    #
    # Default: '.'.
    dir: my-app

    # Custom environment variables to set when invoking node.
    # Invalid environment variables will be ignored.
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - FOO=bar

    # User-tunable subset of the sea-config.json passed to
    # `node --build-sea`. The `output`, `executable`, `useCodeCache`
    # and `useSnapshot` fields are owned by GoReleaser and cannot be
    # set here.
    sea_config:
      # Files baked into the SEA blob and accessible at runtime via
      # sea.getAsset(name).
      assets:
        "icon.png": "./assets/icon.png"
        "schema.json": "./schema.json"

      # Node CLI flags hard-coded into the binary.
      exec_argv:
        - "--max-old-space-size=4096"

      # Whether to silence Node's "experimental SEA" runtime warning.
      # Default: true.
      disable_experimental_sea_warning: true

      # Module system used to evaluate the entrypoint:
      # "commonjs" (default) or "module".
      main_format: commonjs

    # Auto-bundle step. Runs `npm run build` in `dir` before invoking
    # `node --build-sea`, when `package.json` declares a non-empty
    # `scripts.build` entry. Silent skip otherwise. See "Bundling your
    # app" above for details. Dependency installation (`npm ci` and
    # friends) is intentionally not performed — drive it from the
    # global `before` hook.

    # Hooks can be used to customize the final binary, for example to
    # bundle the entrypoint or sign the produced executable.
    #
    # Templates: allowed.
    hooks:
      pre: npx esbuild src/index.js --bundle --platform=node --outfile=dist/bundle.js
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    skip: false
```

The following standard build fields are intentionally **not** supported
by the `node` builder:

- `tool`, `command`, `flags` — the SEA pipeline invokes `node`
  directly with a known set of arguments.

The following template variables are available in the per-target build
context: `.Os`, `.Arch`, `.Goos`, `.Goarch`, `.Target`, `.Name`,
`.Path`, `.Ext`, `.Env.*`. Use them in `main`, `node_version`, `env`,
and the `hooks` recipes.

## Build-tool Node.js

The `--build-sea` subcommand exists only in Node.js ≥ v25.5.0 (built
with LIEF). GoReleaser resolves the build-tool Node in this order:

1. `$GORELEASER_NODE_BUILD_TOOL` — absolute path to a Node binary you
   manage. Must satisfy the `--build-sea` capability probe.
2. `node` on `PATH`, if it satisfies the probe.
3. Auto-download a known-good release into
   `${XDG_CACHE_HOME:-$HOME/.cache}/goreleaser/node/buildtool/<version>/`
   for the host platform. The download (~30 MB) happens once and is
   reused across all subsequent builds.

The capability probe runs `node -p "process.config.variables.node_use_lief"`
and requires it to print `true`; this is the same check the Node.js test
suite uses. Custom Node builds compiled `--without-lief` will not pass
the probe even if their `--version` reports v25.5+.

## Version resolution

The target Node.js version (the binary that becomes the SEA executable)
is resolved in this order:

1. The `node_version` field on the build (recommended for
   reproducible releases).
2. The `engines.node` field in `package.json` (highest matching
   official release).
3. A `.nvmrc` file in the build directory.
4. A `.node-version` file in the build directory.

Either an exact version (`v22.10.0`, `22.10.0`) or a semver range
(`>=22 <23`, `^22`) is accepted. Ranges are resolved against the
nodejs.org release index.

The resolved version must be in the V2-blob-format range understood by
LIEF-emitted SEAs:

- `>= v22.20.0` (back-ported to the v22 LTS line)
- `>= v24.6.0`
- `>= v25.0.0`

Older releases (v18, v20, v22.0–v22.19, v23, v24.0–v24.5) only read the
legacy V1 blob format and will reject the produced binary at runtime.
GoReleaser fails fast in this case with an error pointing at the floor
above.

## Code signing

On macOS, the produced binary is ad-hoc signed in place by the build
when `codesign(1)` is available, which is sufficient for the kernel
loader to accept it locally. **For real distribution you want a real
Developer ID signature plus notarization** — Gatekeeper blocks ad-hoc
signed binaries on user machines. Wire up the
[`signs`](/customization/sign/) and
[`notarize`](/customization/notarize/) pipes to do this on a darwin
runner:

```yaml {filename=".goreleaser.yaml"}
signs:
  - id: macos
    cmd: codesign
    artifacts: binary
    ids: [my-build]
    args: ["--sign", "Developer ID Application: Acme (TEAMID)", "--options=runtime", "--timestamp", "{{ .Path }}"]

notarize:
  macos:
    - ids: [my-build]
      sign:
        certificate: "{{ .Env.MACOS_SIGN_P12 }}"
        password:    "{{ .Env.MACOS_SIGN_PASSWORD }}"
      notarize:
        issuer_id: "{{ .Env.MACOS_NOTARY_ISSUER_ID }}"
        key_id:    "{{ .Env.MACOS_NOTARY_KEY_ID }}"
        key:       "{{ .Env.MACOS_NOTARY_KEY }}"
        wait: true
```

When `codesign(1)` is not available — for example, when cross-compiling
for macOS from a Linux build host — the binary is left unsigned. It is
otherwise well-formed but the macOS kernel will refuse to exec it until
it is signed by the `signs` pipe on a darwin runner.

Windows binaries are also unsigned. SmartScreen and most corporate
allow-listing will block them. Wire up the `signs` pipe with
`signtool.exe` (or your CA's tooling) to sign them after the build
completes:

```yaml {filename=".goreleaser.yaml"}
signs:
  - id: windows
    cmd: signtool
    artifacts: binary
    ids: [my-build]
    args: ["sign", "/fd", "SHA256", "/tr", "http://timestamp.digicert.com", "/td", "SHA256", "/f", "{{ .Env.WIN_CERT_PFX }}", "/p", "{{ .Env.WIN_CERT_PASSWORD }}", "{{ .Path }}"]
```

## License compliance

A SEA binary statically embeds the entire Node.js runtime, which is
distributed under the [MIT license][node-license]. You must include
the Node.js `LICENSE` text alongside your distribution to comply.

The simplest approach is to add it to your archives via the
[`files`](/customization/archive/#packaging) section:

```yaml {filename=".goreleaser.yaml"}
archives:
  - id: my-build
    files:
      - LICENSE
      - src: third_party/NODE_LICENSE
        dst: LICENSE.node
```

Drop a copy of <https://github.com/nodejs/node/blob/main/LICENSE> into
`third_party/NODE_LICENSE` (or wherever) ahead of time. Your own MIT
notice is **not** sufficient — Node's license requires its full text
to ship with the binary.

[node-license]: https://github.com/nodejs/node/blob/main/LICENSE

## Binary size

A SEA binary is the entire Node.js runtime plus your bundled
JavaScript: typically **60–80 MB per platform**, dwarfing the few MB
you would get from a Go binary. This is unavoidable — Node ships V8,
libuv, OpenSSL, ICU, and a full standard library inside every
executable. Plan archive sizes and CDN budgets accordingly.

## Trust model

GoReleaser fetches Node.js binaries from `https://nodejs.org/dist`
over TLS and verifies the SHA-256 of every download against the
matching entry in the per-release `SHASUMS256.txt`. Both the binary
**and** `SHASUMS256.txt` are fetched over TLS only — GoReleaser does
**not** GPG-verify `SHASUMS256.txt.sig` against the Node.js release
team's keyring.

In practice this means GoReleaser trusts the same things `npm`, `nvm`,
and most other Node installers trust: the public PKI and the
nodejs.org CDN. If you need defense against a CDN compromise, fetch
the host binaries yourself (verify `SHASUMS256.txt.sig`) and point the
cache at them, or run GoReleaser through a proxy that does the
verification.

## Environment setup

GoReleaser will not install Node.js for you, but it will auto-download
a build-tool Node.js (≥ v25.5) the first time you build, into its own
cache. Pre-staging that download on a CI runner is optional but speeds
up the first build of a job.

The cache lives at `${XDG_CACHE_HOME:-$HOME/.cache}/goreleaser/node/`
and contains both the build-tool Node and every per-target binary
GoReleaser has fetched. Mount or restore this directory between CI
runs to keep builds offline-friendly.

For users behind a corporate proxy or in a region where nodejs.org is
slow/blocked, set `NODEJS_MIRROR` to point at a mirror (matches the
nvm convention):

```sh
export NODEJS_MIRROR=https://npmmirror.com/mirrors/node
```

The release index, archives, and `SHASUMS256.txt` are all fetched
through the mirror. Trailing slashes on the URL are tolerated.

Transient HTTP failures (5xx, 429, network errors) are retried
automatically with exponential backoff — a single nodejs.org hiccup
will not fail your build.

{{< g_templates >}}
