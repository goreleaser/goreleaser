# Node.js SEA Builder — Remaining Work

Backlog tracking the experimental `node` builder. The legacy ELF/Mach-O/PE
binary surgery (~3.6k LoC) was removed in favour of Node ≥ v25.5's
LIEF-backed `--build-sea` subcommand, which goreleaser now invokes
directly. See
`research/node-sea-builds-is-there-a-well-known-accepted-way.md` for
the migration rationale.

The P1 Configuration/UX and Documentation work is done. What remains is
P2 follow-up — nice-to-have improvements before the builder leaves the
`experimental` tag.

## P2 — Test gaps

### Hooks coverage
**File:** `internal/builders/node/build_test.go`
`pre`/`post` hooks should work because the build pipeline runs them
generically, but there is no explicit coverage.

### Download integrity edge cases
**File:** `internal/nodesea/download_test.go`
Cover: short tar, gzip with trailing garbage, SHASUMS missing the
target line, SHASUMS with wrong digest.

### End-to-end snapshot test
- Run `goreleaser build --snapshot` against
  `internal/static/config.node.yaml` from a CI job that has Node
  ≥ v25.5 cached, exec the produced linux/darwin/windows binaries
  where possible.

## P2 — Polish

### Download progress logging
**File:** `internal/nodesea/download.go`
A 50–100 MB download with no log output looks like a hang.
- `log.WithField("version", v).WithField("target", t).Info("downloading
  node distribution")` before the request, and a "cached" log on the
  hit path.

### Migration guide from `pkg`/`nexe`
**File:** `www/content/customization/builds/builders/node.md`
Most users in this space currently use `vercel/pkg` (now archived) or
`nexe`. A short comparison + migration section would help adoption.

### Plumb retry config from `ctx.Config.Retry`
**File:** `internal/nodesea/download.go`
P5-2 introduced a package-level `defaultRetry` (4 attempts, 1s/30s
backoff) rather than threading the project-wide `Retry` config in,
because `nodesea` deliberately stays decoupled from `*context.Context`.
If users start asking to tune retry per-project, expose a
`SetRetry(config.Retry)` initialiser called from the node builder.

---

## Resolved by the migration to `node --build-sea`

Items eliminated when the legacy injector was deleted (Phase 4) and
addressed in the Phase 5 cleanup:

- `__LINKEDIT` vmsize update on unsign
- `uint32` truncation guards on Mach-O linkedit shift sites
- `uint32` truncation in PE resource RVA / name length
- `FlipSentinel` reading the entire binary into RAM
- Mach-O ad-hoc resigning correctness on Apple Silicon (handled by
  `codesign(1)`; cross-compile leaves the binary unsigned for the
  `signs:` pipe)
- "Stripped → re-injected" pipeline tests (no in-process injection
  remains; `--build-sea` produces the binary atomically)
- File permissions / atomic-write concerns in the inject path
- Hardcoded `sea-config.json` — exposed via `sea_config:` builder field
- "Pipeline integration not validated end-to-end" — `Prepare` runs at
  pipeline start so misconfiguration fails fast
- Tests use synthetic ELF/Mach-O/PE fixtures — fixtures gone, the
  real-Node integration test in `buildsea_test.go` exercises the whole
  flow against an upstream Node release
- `node_version` config field (P5-1)
- Retry on transient nodejs.org failures (P5-2)
- `NODEJS_MIRROR` env var (P5-3)
- `targets.txt` missing platforms — added aix-ppc64, linux-armv7l,
  linux-ppc64le, linux-s390x (P5-4)
- Artifact `Goarm` set on linux-armv7l (P5-5)
- `build.Main` templated (P5-6)
- Documentation: license compliance, binary size, bundling, mirror,
  Windows signtool example, expanded macOS signs+notarize (P5-7)
