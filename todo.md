# Node.js SEA Builder — Remaining Work

Backlog tracking the experimental `node` builder. The pre-migration
audit had 29 items across the legacy ELF/Mach-O/PE binary surgery.
Most were resolved upstream by Node ≥ v25.5's LIEF-backed
`--build-sea` subcommand, which goreleaser now invokes directly. See
`research/node-sea-builds-is-there-a-well-known-accepted-way.md` for
the migration rationale.

Items below are tagged `[P1]` (should-fix before the builder leaves
the `experimental` tag) or `[P2]` (nice-to-have / follow-up).

## P1 — Configuration & UX

### `node_version` config field
**Files:** `internal/builders/node/build.go`, `pkg/config/config.go`
`ResolveVersion(ctx, build.Dir, "")` always passes an empty explicit
version. With `engines.node: ">=20"` the resolved version drifts as
new Node releases ship → builds are not reproducible across runs.
- Add a `node_version` field to the build config (template support),
  thread through `ResolveVersion`, document it, recommend pinning an
  exact version for releases.

### Retry on transient nodejs.org failures
**File:** `internal/nodesea/download.go`
HTTP fetches use a bare `http.Client.Do`. Goreleaser already has
`internal/retryx` that wraps avast/retry-go with `retryx.IsRetriable`
for 5xx/429/network errors. Used by 24+ other pipes.
- Wrap the index, archive, and SHASUMS fetches with
  `retryx.Do(ctx, ctx.Config.Retry, ..., retryx.IsRetriable)`. Reuse
  the existing per-project retry config.

### Mirror / proxy override
**File:** `internal/nodesea/download.go`
`distBaseURL` is hardcoded to `https://nodejs.org/dist`. Users in CN
or behind corporate firewalls need a mirror (e.g.
`https://npmmirror.com/mirrors/node/`).
- Add a config option (e.g. `node_dist_url`, or honour a
  `NODEJS_MIRROR` env var like nvm does). Default unchanged.

### `targets.txt` missing platforms Node ships
**File:** `internal/builders/node/targets.txt`
Currently 6 entries (darwin/linux/windows × x64/arm64). Node also
ships `linux-armv7l`, `linux-ppc64le`, `linux-s390x`, `aix-ppc64`.
Users on those platforms cannot use the builder even though the
upstream binary exists.
- Add the missing targets and matching `convertToGoarch` /
  `convertToGoos` mappings (`armv7l` → `linux/arm` with implicit `v7`).

### Artifact lacks `Goamd64` / `Goarm64` extras
**File:** `internal/builders/node/build.go`
Other builders (Go especially) populate `Goamd64`, `Goarm`, `Goarm64`,
etc. Downstream pipes that template these fields get empty strings for
node artifacts. Probably benign but inconsistent.
- Set the variant fields explicitly (empty string is fine where Node
  has no equivalent variant).

### `build.Main` is not run through templates
**File:** `internal/builders/node/build.go`
Users cannot write `main: dist/{{ .Env.TARGET }}/index.js` or similar.
- Run through `tmpl.New(ctx).Apply(build.Main)` before the
  `os.Stat` and before injecting into the SEA config.

## P1 — Documentation

### License compliance
**File:** `www/content/customization/builds/builders/node.md`
Embedding the Node runtime ships ~80 MB of MIT-licensed code; users
must include the Node `LICENSE` in their distribution to comply.
- Add a "License compliance" section that points at
  https://github.com/nodejs/node/blob/main/LICENSE and recommends
  shipping it with each archive.

### Binary-size warning
**File:** `www/content/customization/builds/builders/node.md`
A SEA binary is 60–80 MB per platform. Users used to Go binaries will
be surprised. Worth calling out alongside the experimental tag.

### Bundling responsibility
**File:** `www/content/customization/builds/builders/node.md`
The builder does not run `npm install`, does not bundle dependencies,
and assumes `main` resolves to a self-contained file. Users with a
typical `dist/index.js + node_modules/` layout will get failures
at SEA-blob generation.
- Add a "Bundling your app" section recommending esbuild / ncc /
  webpack with a single-file output, and provide a `hooks.pre`
  example in `config.node.yaml`.

### Quick start / `goreleaser init --language node`
**File:** `www/content/customization/builds/builders/node.md`
Already wired in `cmd/init.go`. Add a "Quick start" line.

### Windows SmartScreen / macOS Gatekeeper caveats
**File:** `www/content/customization/builds/builders/node.md`
The new docs cover ad-hoc signing and the `signs:` pipe story for
darwin. Add equivalent text for Windows SmartScreen with a sample
`signs:` recipe using `signtool.exe`.

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

---

## Resolved by the migration to `node --build-sea`

Items eliminated outright when the legacy ELF/Mach-O/PE binary surgery
was deleted in favour of Node's LIEF-backed `--build-sea`:

- `__LINKEDIT` vmsize update on unsign
- `uint32` truncation guards on Mach-O linkedit shift sites
- `uint32` truncation in PE resource RVA / name length
- `FlipSentinel` reading the entire binary into RAM
- Mach-O ad-hoc resigning correctness on Apple Silicon (handled by
  `codesign(1)`; cross-compile leaves the binary unsigned for the
  `signs:` pipe)
- "Stripped → re-injected" pipeline tests (no in-process injection
  remains; `--build-sea` produces the binary atomically)
- "Tests use synthetic ELF/Mach-O/PE fixtures" — fixtures gone, the
  real-Node integration test in `buildsea_test.go` exercises the
  whole flow against an upstream Node release
- File permissions / atomic-write concerns in the inject path
- Hardcoded `sea-config.json` — exposed via `sea_config:` builder
  field
- "Pipeline integration not validated end-to-end" — Phase 4 wires the
  builder through `Prepare` so misconfiguration (incompatible target
  Node version) fails fast at the start of the build
