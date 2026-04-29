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
    main: index.js

    # Targets, in nodejs.org/dist format.
    # Default: all of: darwin-arm64, darwin-x64, linux-arm64, linux-x64,
    #                  win-arm64, win-x64.
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

    # Hooks can be used to customize the final binary, for example to
    # run generators or sign the produced executable.
    #
    # Templates: allowed.
    hooks:
      pre: npm run bundle
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    skip: false
```

The following standard build fields are intentionally **not** supported
by the `node` builder:

- `tool`, `command`, `flags` — the SEA pipeline invokes `node`
  directly with a known set of arguments.

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

1. The `engines.node` field in `package.json` (highest matching official
   release).
2. A `.nvmrc` file in the build directory.
3. A `.node-version` file in the build directory.

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
loader to accept it. For real distribution (Gatekeeper, notarization)
you still need to re-sign with a Developer ID via the
[`signs`](/customization/sign/) and [`notarize`](/customization/notarize/)
pipes.

When `codesign(1)` is not available — for example, when cross-compiling
for macOS from a Linux build host — the binary is left unsigned. It is
otherwise well-formed but the macOS kernel will refuse to exec it until
it is signed by the `signs` pipe on a darwin runner.

Windows binaries are unsigned. Wire up the `signs` pipe with
`signtool.exe` (or your CA's tooling) to sign them after the build
completes.

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

## Legacy injector (deprecated)

The pre-`--build-sea` code path — pure-Go ELF/Mach-O/PE binary
surgery driven by `node --experimental-sea-config` — is still
available behind `GORELEASER_NODE_LEGACY_INJECTOR=1` for one release
while the new path bakes in. It will be removed in a subsequent
release. New configurations should use the default flow.

## Environment setup

GoReleaser will not install Node.js for you, but it will auto-download
a build-tool Node.js (≥ v25.5) the first time you build, into its own
cache. Pre-staging that download on a CI runner is optional but speeds
up the first build of a job.

{{< g_templates >}}
