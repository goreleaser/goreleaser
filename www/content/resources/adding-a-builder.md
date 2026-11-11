---
title: "Adding a builder"
weight: 45
---

A builder connects a language's build tool to GoReleaser's build pipeline.
Adding the implementation is only part of the work: the command must load it,
users need an initial configuration, and CI must exercise its toolchain.
This guide follows the existing builders rather than introducing a new plugin
loading mechanism.

## Implement the builder contract

Create a package under `internal/builders/<name>`. The
[`build.Builder` interface](https://github.com/goreleaser/goreleaser/blob/main/pkg/build/build.go)
has three methods:

| Method | Responsibility |
| --- | --- |
| `WithDefaults(config.Build) (config.Build, error)` | Supply tool, command and target defaults; reject unsupported configuration. |
| `Parse(string) (build.Target, error)` | Interpret a target and expose its original string and template fields through `Target`. |
| `Build(*context.Context, config.Build, build.Options) error` | Run the build and register its output artifacts. |

Choose a close existing implementation as a starting point:
[`zig`](https://github.com/goreleaser/goreleaser/blob/main/internal/builders/zig/build.go)
for target-specific executables, or
[`uv`](https://github.com/goreleaser/goreleaser/blob/main/internal/builders/uv/build.go)
for Python distribution artifacts. Do not copy a language's target grammar or
artifact type into a builder with different output semantics.

The build pipe supplies common defaults before calling `WithDefaults`, including
`Binary` and `ID`. `Build.InternalDefaults` distinguishes these from user-supplied
values. For example, the uv builder rejects a user-supplied binary name but allows
the common default because uv determines its own output filenames.

Use the optional interfaces in `pkg/build` only when needed:

- `DependingBuilder` lists external tools for dependency checks.
- `PreparedBuilder` performs preparation before the target builds.
- `ConcurrentBuilder` controls whether target builds can run concurrently.
- `TargetFixer` fills in incomplete target strings.

## Run the tool and record its output

The build pipe creates the output parent directory and runs the configured
pre/post build hooks. The builder receives the output name, path, extension and
parsed target in `build.Options`; it should not implement those hooks again.

The helpers in
[`internal/builders/base`](https://github.com/goreleaser/goreleaser/tree/main/internal/builders/base)
cover common work:

- `ValidateNonGoConfig` rejects Go-only fields for other languages. Pass
  `WithBuildMode` only if the builder supports `buildmode`.
- `TemplateEnv` expands environment entries in order, making earlier entries
  available to later ones.
- `Exec` runs an argument slice with the provided environment, directory and
  context, and handles command output and redaction.
- `ChTimes` applies the configured modification time to an artifact's path.

Follow an existing builder's use of `internal/tmpl` for the tool, flags and
environment. Propagate command and template errors. After a successful build,
add each output to `ctx.Artifacts` with the correct type, path, name and metadata.
Downstream archive and publishing pipes consume these records; writing a file
alone is not sufficient. For example, uv records wheels and source distributions
separately rather than treating them as executable binaries.

## Register and expose it

Use the existing package registration pattern: a default builder instance, a
compile-time `build.Builder` assertion, and `init()` calling
`build.Register("<name>", Default)`.

Registration does not load a package by itself. Add the blank import to
[`internal/pipe/build/build.go`](https://github.com/goreleaser/goreleaser/blob/main/internal/pipe/build/build.go),
where the built-in builder packages are loaded. Test that the build pipe can
resolve the new `builder:` value, not only that the package works in isolation.

If the language should be offered by `goreleaser init`, update these together:

1. Add an example configuration under `internal/static/` and embed it in
   [`config.go`](https://github.com/goreleaser/goreleaser/blob/main/internal/static/config.go).
2. Wire it into the language switch and flag completion in
   [`cmd/init.go`](https://github.com/goreleaser/goreleaser/blob/main/cmd/init.go).
   Add detection in `langDetect` only if an unambiguous project marker exists.
3. Update the static configuration tests and `cmd/init_test.go`, including any
   applicable generated output or detection expectations.

Reuse existing configuration fields where they fit. If new fields are necessary,
update `pkg/config`, their validation and the generated schema through the
repository's schema-generation task. Document the fields and supported targets
on a builder customization page and link the page from the relevant build docs.

## Test the complete integration

Keep the fixture small and commit it under the builder's `testdata/`. Use the
existing builder tests as examples, especially
[`uv/build_test.go`](https://github.com/goreleaser/goreleaser/blob/main/internal/builders/uv/build_test.go)
and [`zig/build_test.go`](https://github.com/goreleaser/goreleaser/blob/main/internal/builders/zig/build_test.go).
Cover:

- Defaults and explicit overrides, invalid targets and unsupported options.
- Target template fields and output artifact metadata.
- The actual tool invocation against a minimal project, including failure paths.
- Registration and initial configuration changes outside the builder package.

Tool-independent tests should run without the external compiler. Follow the
existing tool-availability checks for integration tests, and ensure the tool is
installed in the appropriate jobs in
[`.github/workflows/build.yml`](https://github.com/goreleaser/goreleaser/blob/main/.github/workflows/build.yml).
A locally skipped integration test is not evidence that the build works.

From a complete checkout, run the affected packages first, replacing `<name>`
with your package name:

```sh
go test ./internal/builders/<name> ./internal/pipe/build ./internal/static ./cmd
```

Then follow the [contributing guide](../contributing/), including formatting and
`task ci`. Validate documentation with `task docs:build` and `task docs:test`.
Report the toolchain versions and any skipped or unavailable checks in the PR.
