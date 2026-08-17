---
title: "Telemetry"
weight: 80
---

{{< g_featenterprise >}}

{{< g_version "v2.18-unreleased" >}}

GoReleaser can export [OpenTelemetry][otel] traces of your releases, so you can
see where the time goes: which pipes are slow, which builds dominate, which
upload is flaky, and how it all trends over time.

Traces are always sent to **your** collector, configured through the standard
`OTEL_*` environment variables. GoReleaser never sends anything to us.

## Usage

Telemetry is opt-in, and needs two things:

1. Enable it in your configuration:

   ```yaml {filename=".goreleaser.yaml"}
   telemetry:
     # Whether to export OpenTelemetry traces.
     enabled: true
   ```

2. Point GoReleaser at your collector:

   ```sh
   export OTEL_EXPORTER_OTLP_ENDPOINT="https://otel.mycompany.com"
   export OTEL_EXPORTER_OTLP_HEADERS="authorization=Bearer $TOKEN"
   goreleaser release
   ```

The configuration option exists on purpose: without it, GoReleaser would start
talking to the network whenever a CI runner happens to have `OTEL_*` variables
set for other tooling. You have to ask for it.

If telemetry is enabled but no endpoint is set, GoReleaser warns you, and the
SDK falls back to its own default of `https://localhost:4318`.

## Which commands are traced

The commands that validate a license: `release`, `publish`, `continue`,
`announce`, and `verify`.

`goreleaser build` and `goreleaser release --snapshot` skip the license check
entirely, so they are not traced. Verify your setup with a real release.

## Environment variables

GoReleaser uses the [standard OpenTelemetry environment variables][otelenv].
The most useful ones:

| Variable                      | Description                                                    |
| ----------------------------- | -------------------------------------------------------------- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint, e.g. `https://otel.mycompany.com`.         |
| `OTEL_EXPORTER_OTLP_HEADERS`  | Extra headers, e.g. `authorization=******`.                    |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` (default) or `grpc`.                           |
| `OTEL_SERVICE_NAME`           | Service name. Defaults to `goreleaser`.                        |
| `OTEL_RESOURCE_ATTRIBUTES`    | Extra resource attributes, e.g. `team=platform,env=prod`.      |
| `TRACEPARENT`                 | If set, the release is nested inside your CI pipeline's trace. |

The `_TRACES_` variants (e.g. `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`) take
precedence over the generic ones, as per the specification.

{{< callout type="info" >}}
These are read from the actual process environment, not from the `env` section
of your configuration file: the OpenTelemetry SDK reads them directly.
{{< /callout >}}

## What you get

One trace per run, with:

- a **root span** named after the command,
- a **child span per pipe** (`building binaries`, `archives`, `publishing`…),
- a **span per publisher** inside `publishing` (`docker images`, `blobs`,
  `homebrew cask`…),
- a **span per build target** and per **docker image**, which is usually where
  the time actually goes.

Releasing a small Go project looks roughly like this:

```text
release                                 1843ms
├─ getting and validating git state      177ms
├─ setting defaults                       29ms
├─ building binaries                     451ms
│  ├─ build                              416ms  goreleaser.build.target=linux_amd64_v1
│  ├─ build                              253ms  goreleaser.build.target=darwin_arm64_v8.0
│  └─ build                              428ms  goreleaser.build.target=windows_amd64_v1
├─ generating changelog                   11ms
├─ archives                              166ms
├─ calculating checksums                   1ms
└─ publishing                           1008ms
   └─ docker images                     1007ms
```

Builds run in parallel, so the `build` spans overlap and can add up to more than
their parent.

Span names are kept low-cardinality on purpose: what varies from one build to
the next lives in the attributes, so your backend can aggregate `build` across
targets, projects, and runs.

Spans are annotated with the [CI/CD semantic conventions][semconv], so they
should light up in your existing dashboards without any mapping:

| Attribute                       | Where       | Example                                         |
| ------------------------------- | ----------- | ----------------------------------------------- |
| `cicd.pipeline.name`            | root        | your project name                               |
| `cicd.pipeline.run.id`          | root        | the trace ID                                    |
| `cicd.pipeline.result`          | root        | `success`, `failure`, `timeout`, `cancellation` |
| `cicd.pipeline.run.url.full`    | root        | the release URL                                 |
| `vcs.repository.url.full`       | root        | your repository URL                             |
| `vcs.ref.head.name`             | root        | the branch                                      |
| `vcs.ref.head.revision`         | root        | the commit                                      |
| `goreleaser.project_version`    | root        | `1.2.3`                                         |
| `goreleaser.artifacts`          | root        | how many artifacts were produced                |
| `goreleaser.snapshot`           | root        | whether it is a snapshot                        |
| `goreleaser.nightly`            | root        | whether it is a nightly                         |
| `cicd.pipeline.task.name`       | pipe        | `publishing`                                    |
| `cicd.pipeline.task.run.result` | pipe        | `success`, `failure`, `skip`…                   |
| `goreleaser.build.id`           | build span  | your build ID                                   |
| `goreleaser.build.builder`      | build span  | `go`, `rust`, `zig`…                            |
| `goreleaser.build.target`       | build span  | `linux_amd64_v1`                                |
| `goreleaser.docker.id`          | docker span | your docker ID                                  |
| `goreleaser.docker.platform`    | docker span | `linux/amd64`                                   |
| `goreleaser.docker.platforms`   | docker span | `linux/amd64`, `linux/arm64` (Docker v2)        |

Skipped pipes are recorded as `skip` rather than as errors, so `--skip=publish`
runs don't pollute your error rates.

Every span also carries the usual resource attributes: `service.name`
(`goreleaser` unless you override it), `service.version` (the GoReleaser version
that produced the release), the host, and the SDK details.

When a release fails, the failing span and the root span are both marked as
errors, and the trace is still exported — the broken release is exactly the one
you want to inspect afterwards.

## Failure handling

Telemetry never fails a release. If the collector is unreachable or rejects the
data, GoReleaser warns once and carries on — the release is not held hostage to
your observability stack. The final flush is bounded, so a dead collector can't
hang the end of your pipeline.

A malformed `OTEL_RESOURCE_ATTRIBUTES` doesn't disable tracing either: the
attributes that did parse are kept, and GoReleaser warns about the rest.

[otel]: https://opentelemetry.io
[otelenv]: https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/
[semconv]: https://opentelemetry.io/docs/specs/semconv/cicd/
