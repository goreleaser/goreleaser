# Node.js SEA Builder — Audit Findings

Backlog produced from a 5-agent audit of the `nodejs-sea` branch
(diff vs `main` adds ~4 100 lines across `internal/nodesea/`,
`internal/builders/node/`, `cmd/init.go`, `internal/static/`,
`pkg/config/config.go`, and `www/content/`).

Code currently compiles and `go test ./internal/nodesea/...
./internal/builders/node/...` passes — these are pre-merge issues, not
regressions.

Items are tagged `[P0]` (must-fix before merge), `[P1]` (should-fix
before announce), `[P2]` (nice-to-have / follow-up).

---

## P0 — Bugs and security issues

### Mach-O `__LINKEDIT` vmsize not updated when stripping signature
**Status:** ✅ Fixed in the Mach-O fuse refactor (`macho.go:unsignMachOBytes`).
Now writes both `vmsize` (offset +32) and `filesize` (offset +48) when
shrinking `__LINKEDIT`. `TestUnsignMachOBytes/strips trailing signature
and updates linkedit vmsize` is the regression guard.

### File permissions are not preserved across inject/unsign/codesign
**Status:** ✅ Not applicable for the nodesea pipeline. After the
public-API collapse, the only call path is
`Build` → `downloadHost` (writes cache at `0o755`) → format-specific
`build*` (rewrites at `0o755`). Every link in the chain is `0o755` by
construction, matching every other goreleaser builder. The injectors
are now package-private, so there is no library consumer whose perms
could be dropped.

### Cache writes are not atomic on Linux/macOS
**Status:** ✅ Fixed in `download.go:extractNodeFromTarGz`. Extracts to a
sibling tempfile and renames over `dst` on success; failed extracts now
clean up the tempfile and never leave a partial binary at the canonical
cache path. (Per-version lockfile for concurrent goroutines remains a P2
nice-to-have.)

### No GPG verification of `SHASUMS256.txt`
**Status:** ✅ Documented as a deliberate trust-model choice. See
`www/content/customization/builds/builders/node.md` ("Trust model"
section). We trust TLS + nodejs.org CDN, the same anchors `npm`,
`nvm`, and most Node installers trust. Adding GPG would require
embedding and rotating the Node release-team keyring, which we judge
not worth the maintenance cost.

### Tar extraction does not defend against path traversal
**Status:** ✅ Not applicable. The extractor only matches one fully
qualified entry name (`node-vX.Y.Z-os-arch/bin/node`) we built
ourselves, and the destination path is the caller-controlled `dst`
— `h.Name` is never joined onto a filesystem path, so there is no
zip-slip surface to defend.

### `uint32` truncation in Mach-O linkedit shifts
**Status:** ✅ Fixed in the Mach-O fuse refactor (`macho.go:injectMachOBytes`,
`macho.go:shiftLinkeditFileOffsets`). All three sites
(`section_64.offset`, `LC_DYLD_CHAINED_FIXUPS.dataoff`, the linkedit
shift loop) now compute in `uint64` and return `ErrNotSupported` if the
result would exceed `math.MaxUint32`.

### `uint32` truncation in PE resource RVA / name length
**Status:** ✅ Fixed in `inject_pe.go:serialize`. Layout offsets and
`va + dataOff` are now computed in `uint64` and validated against
`math.MaxUint32`; resource names exceeding `0xFFFF` runes are rejected
with `ErrNotSupported`.

---

## P1 — Correctness, design, and missing features

### No way to pin the Node.js version in `.goreleaser.yaml`
**Files:** `internal/builders/node/build.go:203`,
`pkg/config/config.go` (`Build` struct)
`ResolveVersion(ctx, build.Dir, "")` always passes an empty explicit
version. With `engines.node: ">=20"` the resolved version drifts as
new Node releases ship → builds are not reproducible across runs.
- Add a `node_version` (or `node: { version: ... }`) field to the
  builder config, thread it through to `ResolveVersion`, document it,
  and recommend pinning an exact version for releases.

### Hardcoded SEA config — `useSnapshot`, `useCodeCache`, `assets` unreachable
**File:** `internal/builders/node/build.go:218-222`
The generated `sea-config.json` only sets `main`, `output`, and
`disableExperimentalSEAWarning`. Users cannot opt into V8 snapshot,
code cache, or asset embedding without forking goreleaser.
- Expose a `sea_config: map[string]any` (or typed struct) field;
  shallow-merge user values over the defaults.

### No retry on transient nodejs.org failures
**File:** `internal/nodesea/download.go:151-204`
HTTP fetches use a bare `http.Client.Do`. Goreleaser already has
`internal/retryx` (used by 24+ other pipes) that wraps avast/retry-go
with `retryx.IsRetriable` for 5xx/429/network errors.
- Wrap the index, archive, and SHASUMS fetches with
  `retryx.Do(ctx, ctx.Config.Retry, ..., retryx.IsRetriable)`. Reuse
  the existing per-project retry config.

### No mirror / proxy override
**File:** `internal/nodesea/download.go:70`
`distBaseURL` is hardcoded to `https://nodejs.org/dist`. Users in CN
or behind corporate firewalls need a mirror (e.g.
`https://npmmirror.com/mirrors/node/`).
- Add a config option (e.g. `node_dist_url`, or honour a
  `NODEJS_MIRROR` env var like nvm does). Default unchanged.

### `targets.txt` missing platforms that Node actually ships
**File:** `internal/builders/node/targets.txt`
Currently 6 entries (darwin/linux/windows × x64/arm64). Node also
ships `linux-armv7l`, `linux-ppc64le`, `linux-s390x`, `aix-ppc64`.
Users on those platforms cannot use the builder even though the
upstream binary exists.
- Add the missing targets and matching `convertToGoarch` /
  `convertToGoos` mappings (`armv7l` → `linux/arm` with implicit `v7`).

### Validation rejects valid-but-unlisted upstream targets
**Files:** `internal/builders/node/targets.go:77-86`,
`internal/builders/node/build.go:107-111`
`isValid` checks against the embedded `targets.txt`, so any target not
in the static list is rejected at `WithDefaults` time even if Node
publishes it. Couples the builder release cadence to Node's.
- Once `targets.txt` is complete, this is fine. Otherwise consider
  letting the download fail naturally with a clear error.

### Artifact lacks `Goamd64` / `Goarm64` extras
**File:** `internal/builders/node/build.go:134-147`
Other builders (Go especially) populate `Goamd64`, `Goarm`, `Goarm64`,
etc. Downstream pipes that template these fields get empty strings for
node artifacts. Probably benign but inconsistent.
- Set the variant fields explicitly (empty string is fine where Node
  has no equivalent variant).

### `build.Main` is not run through templates
**File:** `internal/builders/node/build.go:199, 219`
Users cannot write `main: dist/{{ .Env.TARGET }}/index.js` or similar.
- Run through `tmpl.New(ctx).Apply(build.Main)` before the
  `os.Stat` and before injecting into the SEA config.

### `FlipSentinel` reads the entire ~70 MB binary into RAM
**File:** `internal/nodesea/sentinel.go:39`
`io.ReadAll` to flip a single byte. Wasteful per-target.
- Mach-O path now flips the sentinel in-memory inside `BuildMachO`
  (no separate I/O round-trip).
- ELF/PE paths still call the file-based `FlipSentinel` after their
  own writes; could be folded into their inject paths similarly.

### No Windows re-signing path
**File:** `internal/nodesea/unsign_pe.go` (and missing counterpart)
Mach-O has `codesign_macho.go` for ad-hoc resigning. PE has nothing —
the output is left unsigned, which means SmartScreen / corp policy
will block it. Goreleaser's `signs` pipe can post-process, but this is
not surfaced in docs.
- Either provide a `signtool.exe` shim (when present), or document the
  expected `signs:` recipe end-to-end.

### Ad-hoc signing on macOS arm64 may not be enough for Gatekeeper
**File:** `internal/nodesea/macho.go:adHocSignFile`,
`www/content/customization/builds/builders/node.md:110-114`
Ad-hoc signatures let the kernel exec the binary on Apple Silicon,
but Gatekeeper (and notarization) require a real Developer ID.
- Document the expected `signs:` + `notarize:` flow. Optionally accept
  a signing identity argument so users can plug in a real cert.

### Pipeline integration not validated end-to-end
**File:** `internal/nodesea/pipeline.go`
Worth confirming: `String()`, `Skip(ctx)`, ordering relative to
`build`, registration in `internal/pipeline/pipeline.go`,
`pkg/healthcheck/healthcheck.go` listing, and `goreleaser check`
output for a node config.
- Add an end-to-end test that runs `goreleaser build --snapshot`
  against `internal/static/config.node.yaml`.

### `goreleaser check` rejects the example node config
**File:** `internal/static/config.node.yaml`
Reported by audit: running `goreleaser check -f
internal/static/config.node.yaml` against the current `main` build
errors with `invalid builder: node`. Worth re-confirming on this
branch — if it still fails, the static config tests aren't actually
exercising it.
- Re-run on this branch; if it fails, fix the wiring (probably a
  missing `init()` register call or pipeline registration).

---

## P1 — Documentation and UX

### Missing license / attribution warning
**File:** `www/content/customization/builds/builders/node.md`
Embedding the Node runtime ships ~80 MB of MIT-licensed code; users
must include the Node `LICENSE` in their distribution to comply.
- Add a "License compliance" section that points at
  https://github.com/nodejs/node/blob/main/LICENSE and recommends
  shipping it with each archive.

### Missing binary-size warning
**File:** `www/content/customization/builds/builders/node.md`
A SEA binary is 60–80 MB per platform. Users used to Go binaries will
be surprised. Worth calling out alongside the experimental tag.

### Network requirement is buried
**File:** `www/content/customization/builds/builders/node.md:24-27`
Only mentioned in a procedural step. CI/airgapped users will hit this
hard. Promote to a top-level "Environment setup" section, document the
cache path (`$XDG_CACHE_HOME/goreleaser/node/`), and explain how to
pre-populate the cache in airgapped environments.

### Windows SmartScreen / macOS arm64 caveats not documented
**File:** `www/content/customization/builds/builders/node.md:110-114`
Doc currently warns about macOS unsigned binaries refusing to run.
Add equivalent text for Windows SmartScreen and the Apple Silicon
Gatekeeper / notarization requirement (see also: ad-hoc signing item
above).

### Bundling responsibility not stated
**File:** `www/content/customization/builds/builders/node.md`
The builder does not run `npm install`, does not bundle dependencies,
and assumes `main` resolves to a self-contained file. Users with a
typical `dist/index.js + node_modules/` layout will get failures
at SEA-blob generation.
- Add a "Bundling your app" section recommending esbuild / ncc /
  webpack with a single-file output, and provide a `hooks.pre`
  example in `config.node.yaml`.

### `g_version` shortcode missing from doc
**File:** `www/content/customization/builds/builders/node.md`
Other builder docs include `{{< g_version "v2.X" >}}` so the docs
site renders the introduction version badge.

### `.Target` template variable is undocumented
**File:** `www/content/customization/builds/builders/node.md`
Set in `build.go:140`; the bun docs explicitly call out the
equivalent. Add a one-liner.

### `goreleaser init --language node` is undocumented
**File:** `www/content/customization/builds/builders/node.md`
Already wired in `cmd/init.go`. Add a "Quick start" line.

---

## P2 — Test gaps

### Tests use synthetic ELF/Mach-O/PE fixtures, not real Node binaries
**Files:** `internal/nodesea/{elf,macho,pe}_fixture_test.go`
Hand-rolled fixtures have a single PT_LOAD / two segments / one
section. Real Node has dozens of load commands, large symtabs,
chained fixups, version-info resource trees, etc. Several of the
truncation/overflow bugs above are invisible against these fixtures.
- Add an opt-in integration test (`-tags integration` or a `TestMain`
  guard) that downloads a real Node binary into a cached temp dir,
  runs the full strip → inject → flip-sentinel pipeline, re-parses
  with `debug/elf` / `debug/macho` / `debug/pe`, and (on host-matching
  platforms) actually executes it with
  `node --print "process.isSEA"`.

### No cross-arch test coverage
**Files:** `inject_macho_test.go` (fixture is ARM64 only),
`inject_elf_test.go` (fixture is x86_64 only)
Apple Silicon uses 16 KB pages vs 4 KB on x86_64; the page-alignment
math is currently exercised only for one of the two combinations.
- Parameterise fixtures over `(cputype, pageSize)` and assert
  alignment for each.

### No "stripped → re-injected" pipeline test
**Files:** all `inject_*_test.go`
Real flow is `UnsignMachO/UnsignPE` → `InjectMachO/InjectPE` → user
re-signs. Tests only run inject on a fresh fixture. Order-of-ops bugs
between the two passes (e.g. stale offsets after truncation) won't
surface.
- Add a combined test per format.

### No "binary actually loadable" assertion
**Files:** all `inject_*_test.go`
Tests re-parse with `debug/{elf,macho,pe}` (good) but don't exec the
output. A binary can parse and still ENOEXEC.
- Where the host arch matches, attempt a `os/exec.Command` (with a
  `--version`-equivalent flag) and assert the loader doesn't reject
  it.

### No tar extraction security test
**File:** `internal/nodesea/download_test.go`
See P0 path-traversal item — once the guard is added, lock it in with
a test that feeds in a tar with `../etc/passwd` and asserts rejection.

### No hooks test for the node builder
**File:** `internal/builders/node/build_test.go`
`pre`/`post` hooks should work because the build pipeline runs them
generically, but there's no explicit coverage.

### No download/integrity test for malformed archives
**File:** `internal/nodesea/download_test.go`
Cover: short tar, gzip with trailing garbage, SHASUMS missing the
target line, SHASUMS with wrong digest.

---

## P2 — Polish

### Add download progress logging
**File:** `internal/nodesea/download.go:94-147`
A 50–100 MB download with no log output looks like a hang.
- `log.WithField("version", v).WithField("target", t).Info("downloading
  node distribution")` before the request, and a "cached" log on the
  hit path.

### Loosen "thin Mach-O only" error message
**File:** `internal/nodesea/unsign_macho.go:52-53`
We reject Fat/Universal binaries with `ErrNotSupported`. Users who
hit this typically don't know what `lipo -thin` is.
- Wrap the error with: "use `lipo -thin <arch> <input> -output <out>`
  to extract a single architecture first".

### Document or remove debug-data LC stripping
**File:** `internal/nodesea/unsign_macho.go:24-26`
`unsignMachO` removes `LC_FUNCTION_STARTS`, `LC_DATA_IN_CODE`, and
`LC_SOURCE_VERSION` to free header space. This is a deliberate
trade-off (loses Instruments / dtrace symbol coverage and crash-report
source version) — call it out in the doc so users with profiling
needs know.

### Migration guide from `pkg`/`nexe`
**File:** `www/content/customization/builds/builders/node.md`
Most users in this space currently use `vercel/pkg` (now archived) or
`nexe`. A short comparison + migration section would help adoption.

---

## Summary by area

| Area                                | P0 | P1 | P2 |
|-------------------------------------|----|----|----|
| Mach-O signing/unsigning            |  ~~1~~ ✅ |  1 |  1 |
| Inject (ELF / Mach-O / PE)          |  ~~3~~ → 1 (PE only) |  0 |  0 |
| Download / integrity                |  3 |  2 |  2 |
| Builder integration                 |  0 |  4 |  0 |
| Public API (config struct)          |  0 |  2 |  0 |
| Docs / UX                           |  0 |  6 |  1 |
| Tests                               |  0 |  0 |  6 |
| **Total open**                      |  **4** | **15** | **10** |

P0 items should block merge. P1 items can land in follow-ups but
should be done before the feature loses its `experimental` tag.

### Resolved by the Mach-O fuse refactor
- Mach-O `__LINKEDIT` vmsize update on unsign.
- `uint32` truncation guards on the three Mach-O linkedit shift sites.
- File permissions handling consolidated for the Mach-O path
  (atomic tempfile + rename in `BuildMachO`).
- `FlipSentinel` no longer reads the whole binary on the Mach-O path
  (now an in-memory step inside `BuildMachO`).
- Public API collapsed from 10 exports
  (`PrepareHost`/`Inject`/`Unsign`/`InjectMachO`/`UnsignMachO`/
  `AdHocSignMachO`/`InjectELF`/`InjectPE`/`UnsignPE`/`FlipSentinel`)
  down to a single `Build(ctx, version, target, outPath, blob) error`.

