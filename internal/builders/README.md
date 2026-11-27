# Adding a builder

A builder connects a language's build tool to GoReleaser. Use
[Zig](zig/build.go) as an example for target-specific executables, or
[uv](uv/build.go) for Python distributions.

## Implement the builder

Create a package under `internal/builders/<name>` that implements
[`build.Builder`](../../pkg/build/build.go).

| Method | Responsibility |
| --- | --- |
| `WithDefaults` | Set tool, command, and target defaults. Reject unsupported configuration. |
| `Parse` | Interpret a target and expose its original string and template fields. |
| `Build` | Run the tool and register its output artifacts. |

The [build pipe](../pipe/build/build.go) supplies common defaults such as
`Binary` and `ID` before calling `WithDefaults`. Use `Build.InternalDefaults`
to distinguish those from user-supplied values. The pipe also creates the
output directory and runs pre- and post-build hooks. Read the output path
and parsed target from `build.Options` in the builder.

The [base helpers](base/) can validate non-Go configuration, expand
environment entries, run the build command, and apply artifact timestamps.
Use [`internal/tmpl`](../tmpl/) for templated configuration and return template
or command errors. Register each successful output in `ctx.Artifacts` with
its path, name, type, and metadata. Downstream pipes rely on those records.
For example, uv registers wheels and source distributions separately.

Use the optional interfaces in [`pkg/build`](../../pkg/build/build.go) when
you need to declare external dependencies (`DependingBuilder`), prepare
before builds (`PreparedBuilder`), control target concurrency
(`ConcurrentBuilder`), or complete target strings (`TargetFixer`).

## Register the builder

1. Follow the existing pattern: declare a default instance and a compile-time
   `build.Builder` assertion, then call `build.Register("<name>", Default)`
   from `init()`.
2. Add a blank import for the package in the
   [build pipe](../pipe/build/build.go) so its `init()` runs.
3. Add the builder name to the `jsonschema` enum tag on `Build.Builder` in
   [`pkg/config/config.go`](../../pkg/config/config.go). Run
   `task schema:generate` and include the updated
   [`www/static/schema.json`](../../www/static/schema.json). Do this for every
   builder, including ones that reuse all existing configuration fields.
   The [schema command](../../cmd/schema.go) reflects the configuration
   type, so a name registered at runtime is accepted by schema validators
   only after it is added to the enum.

Validate a minimal configuration such as the one below, replacing `example`
with the registered name. Check it against the schema emitted by
`go run . schema` and against `www/static/schema.json`.

```yaml
version: 2
builds:
  - builder: example
```

If the builder needs new settings, add their validation and document them
and the supported targets in the build documentation.

## Add an init template when applicable

To offer the language through `goreleaser init`, embed a template in
[`internal/static/config.go`](../static/config.go), wire it into the language
switch and flag completion in [`cmd/init.go`](../../cmd/init.go), and update
the corresponding tests. Add detection in `langDetect` if the language has
an unambiguous project marker.

## Test the integration

Keep a minimal fixture in the builder's `testdata/` directory. Cover
defaults, explicit overrides, invalid targets and options, tool execution
and errors, and artifact metadata. Verify that the build pipe can resolve
the registered builder name. See the [Zig](zig/build_test.go) and
[uv](uv/build_test.go) tests for examples.

Keep tests that do not invoke the tool runnable without it. For integration
tests, follow existing tool-availability checks and install the tool in the
appropriate jobs in [`.github/workflows/build.yml`](../../.github/workflows/build.yml).

Run the affected packages from the repository root, substituting the new
package name for `example`.

```sh
go test ./internal/builders/example ./internal/pipe/build ./internal/static ./cmd
```

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for the full checks to run.
