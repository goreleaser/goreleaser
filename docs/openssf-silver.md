# OpenSSF Best Practices — Silver evidence matrix

Working notes for [issue #7267](https://github.com/goreleaser/goreleaser/issues/7267)
toward the [Silver badge](https://www.bestpractices.dev/en/projects/5420/silver)
([criteria](https://www.bestpractices.dev/en/criteria/1)).

The project already holds the
[Passing badge](https://www.bestpractices.dev/projects/5420).

**Status legend**

| Status | Meaning |
| --- | --- |
| Met | Public evidence exists; safe to mark Met with the listed URL once reviewed |
| Likely Met | Strong evidence, but needs a quick maintainer confirmation before marking Met |
| Gap | Missing or insufficient public evidence |
| N/A | Criterion does not apply; justification included |
| Maintainer-only | Cannot be closed from docs alone (access, succession, private process facts) |

Do **not** mark badge-site criteria Met without reviewing the linked evidence.
Do **not** invent maintainer-only facts.

## Silver MUST criteria

| ID | Criterion | Status | Evidence URL | Notes |
| --- | --- | --- | --- | --- |
| `achieve_passing` | Project has Passing badge | Met | https://www.bestpractices.dev/projects/5420 | Already Passing |
| `contribution_requirements` | Contribution requirements documented | Met | https://goreleaser.com/resources/contributing/ | Coding standards + testing policy in `CONTRIBUTING.md` / site mirror |
| `governance` | Governance model documented | Likely Met | https://github.com/goreleaser/goreleaser/blob/main/GOVERNANCE.md | **Draft** — BDFL model based on public maintainers page; needs maintainer confirmation |
| `code_of_conduct` | Code of conduct in a standard location | Met | https://github.com/goreleaser/.github/blob/main/CODE_OF_CONDUCT.md | Linked from contributing docs |
| `roles_responsibilities` | Key roles and responsibilities documented | Likely Met | https://goreleaser.com/resources/maintainers/ | Extended with role language from existing public credits; confirm with maintainers |
| `access_continuity` | Project can continue if one person is lost | Maintainer-only | — | Needs public statement of who else can cut issues/PRs/releases and hold required access |
| `documentation_roadmap` | Roadmap for ≥ next year | Likely Met | https://goreleaser.com/resources/roadmap/ | **Draft** from public v3 discussion/milestones; uncertain items marked as proposals |
| `documentation_architecture` | Architecture / high-level design docs | Met | https://goreleaser.com/getting-started/how-it-works/ | `www/content/getting-started/how-it-works.md` |
| `documentation_security` | What users can / cannot expect for security | Met | https://goreleaser.com/resources/security/ | “Security expectations” section + reporting/response |
| `documentation_quick_start` | Quick start for new users | Met | https://goreleaser.com/getting-started/quick-start/ | `www/content/getting-started/quick-start.md` |
| `documentation_current` | Docs kept consistent with current software | Likely Met | https://goreleaser.com/ | Docs live in-repo (`www/`) and ship with releases; deprecations tracked |
| `documentation_achievements` | Achievements / badge linked from front page | Met | https://github.com/goreleaser/goreleaser/blob/main/README.md | OpenSSF Best Practices badge in `README.md` |
| `sites_password_security` | Password storage on project sites | N/A | — | Auth is via GitHub/third-party hosts; project sites do not store end-user passwords |
| `maintenance_or_update` | Maintain older versions or document upgrade path | Likely Met | https://goreleaser.com/resources/deprecations/ | SemVer releases + deprecation/migration docs; confirm support window wording if needed |
| `report_tracker` | Issue tracker for individual issues | Met | https://github.com/goreleaser/goreleaser/issues | GitHub Issues |
| `vulnerability_report_credit` | Credit vulnerability reporters (unless anonymous) | Met | https://github.com/goreleaser/goreleaser/security/advisories | Credit policy in `SECURITY.md`; recent advisories list reporters |
| `vulnerability_response_process` | Documented vulnerability response process | Met | https://goreleaser.com/resources/security/ | `SECURITY.md`, site mirror, and `INCIDENT_RESPONSE.md` |
| `coding_standards` | Identify coding style and require compliance | Met | https://goreleaser.com/resources/contributing/ | Effective Go + gofumpt + `.golangci.yaml` |
| `coding_standards_enforced` | Automatically enforce coding style | Met | https://github.com/goreleaser/goreleaser/blob/main/.github/workflows/build.yml | `golangci-lint` job; config in `.golangci.yaml` |
| `build_standard_variables` | Honor compiler/linker env vars for native builds | Likely Met | https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program | `go build` honors `CGO_*` / related env vars when CGO is used; confirm justification text on badge site |
| `build_non_recursive` | No recursive subdirectory builds with cross-deps | Met | https://github.com/goreleaser/goreleaser/blob/main/Taskfile.yml | Single-module `go build` / Task; not a recursive Make tree |
| `build_repeatable` | Repeatable bit-for-bit builds | Likely Met | https://github.com/goreleaser/goreleaser/blob/main/go.sum | Modules pinned via `go.mod`/`go.sum`; full bit-identical release reproducibility may still need badge wording |
| `installation_common` | Easy install/uninstall via common conventions | Met | https://goreleaser.com/getting-started/install/ | Homebrew, apt/yum, Scoop, npm, containers, etc. |
| `installation_standard_variables` | Honor DESTDIR / install location conventions | N/A | — | End-user install is via package managers / prebuilt artifacts, not a DESTDIR-style source installer |
| `installation_development_quick` | Quick contributor install of build/test env | Met | https://goreleaser.com/resources/contributing/ | `task` + Go; `task ci` |
| `external_dependencies` | List external deps in computer-processable form | Met | https://github.com/goreleaser/goreleaser/blob/main/go.mod | `go.mod` / `go.sum` |
| `dependency_monitoring` | Monitor deps for known vulnerabilities | Met | https://github.com/goreleaser/goreleaser/blob/main/.github/workflows/govulncheck.yml | Also `depsreview.yaml`, `grype.yml`, `sbom-scan.yml`, Dependabot |
| `updateable_reused_components` | Easy to identify/update reused components | Met | https://github.com/goreleaser/goreleaser/blob/main/go.mod | Go modules + Dependabot (`.github/dependabot.yml`) |
| `automated_integration_testing` | Automated tests on check-in with pass/fail report | Met | https://github.com/goreleaser/goreleaser/blob/main/.github/workflows/build.yml | CI on PRs/pushes |
| `regression_tests_added50` | Regression tests for ≥50% of bugs fixed (6 months) | Likely Met | https://github.com/goreleaser/goreleaser/blob/main/.github/PULL_REQUEST_TEMPLATE.md | PR template asks for tests; formal SHOULD/MUST policy now in contributing docs — maintainer should spot-check recent bugfix |
| `test_statement_coverage80` | ≥80% statement coverage | Met | https://codecov.io/gh/goreleaser/goreleaser | Codecov reports ~91% on `main` (verify current badge when submitting) |
| `test_policy_mandated` | Formal policy: major new functionality MUST add tests | Met | https://goreleaser.com/resources/contributing/ | Testing policy section |
| `tests_documented_added` | Change instructions include adding tests | Met | https://goreleaser.com/resources/contributing/ | Same contributing docs |
| `warnings_strict` | Maximally strict warnings where practical | Met | https://github.com/goreleaser/goreleaser/blob/main/.golangci.yaml | golangci-lint enabled in CI |
| `implement_secure_design` | Implement secure design principles | Likely Met | https://goreleaser.com/resources/assurance-case/ | Argument in assurance case; grounded in `THREAT_MODEL.md` + CI controls |
| `crypto_weaknesses` | Default security mechanisms avoid weak crypto | Likely Met | https://goreleaser.com/getting-started/install/ | Release verification uses Sigstore/cosign + SHA-256; HTTPS via Go stdlib |
| `crypto_credential_agility` | Credentials/keys stored separately, replaceable | Met | https://goreleaser.com/resources/security/ | Tokens/keys via env / CI secrets, not compiled in |
| `crypto_certificate_verification` | TLS certificate verification by default | Likely Met | https://goreleaser.com/resources/assurance-case/ | Relies on Go `net/http` / TLS defaults for outbound HTTPS |
| `crypto_verification_private` | Verify certs before sending private HTTP data | Likely Met | https://goreleaser.com/resources/assurance-case/ | Same as above for API/token traffic |
| `signed_releases` | Cryptographically signed releases + verify docs | Met | https://goreleaser.com/getting-started/install/ | Cosign + checksums in OSS install docs; CI verify job in `.github/workflows/release.yml` |
| `input_validation` | Validate untrusted inputs (allowlist) | Likely Met | https://goreleaser.com/resources/assurance-case/ | Config/CLI validation and typed config; see assurance case |
| `assurance_case` | Assurance case (threats, trust boundaries, arguments) | Met | https://goreleaser.com/resources/assurance-case/ | Complements root `THREAT_MODEL.md` |
| `static_analysis_common_vulnerabilities` | Static analysis for common vulns | Met | https://github.com/goreleaser/goreleaser/blob/main/.github/workflows/codeql.yml | Also Semgrep (in `build.yml`), govulncheck, gitleaks, Scorecard |
| `dynamic_analysis_unsafe` | Dynamic analysis if memory-unsafe languages used | N/A | — | GoReleaser is primarily Go (memory-safe); no C/C++ product surface requiring this |

## Related SHOULD / SUGGESTED (not blocking MUST matrix)

| ID | Status | Notes |
| --- | --- | --- |
| `dco` | Gap | No DCO/CLA workflow documented in-repo today |
| `bus_factor` | Maintainer-only / Likely Met | Multiple named maintainers publicly, but release/admin continuity is maintainer-confirmed |
| `version_tags_signed` | Gap | Recent `v*` tags observed as lightweight (unsigned) commit tags; release *artifacts* are signed via cosign |
| `hardening` | Likely Met | Go memory safety, least-privilege workflow tokens, fuzz tasks — document on badge if desired |
| `crypto_algorithm_agility` | Likely Met / N/A | Signing delegated to Sigstore/cosign; not a custom crypto suite |
| `crypto_used_network` / `crypto_tls12` | Likely Met | Outbound HTTPS to forges/registries via Go TLS |

## Maintainer-only follow-ups (Carlos)

1. Confirm or revise **`GOVERNANCE.md`** (decision model / BDFL wording).
2. Confirm or revise **`ROADMAP.md` / site roadmap** (what is commitment vs proposal).
3. Publish an **access continuity** statement (who can issue/PR/release if the BDFL is unavailable; where critical secrets/DNS/org ownership live — without publishing secret material).
4. Decide whether to adopt **DCO** (`Signed-off-by`) or document another contribution license mechanism.
5. Optionally start **signing annotated git tags** for `version_tags_signed`.
6. Spot-check **regression test rate** for the last six months of bug fixes before marking `regression_tests_added50`.
7. After URLs are live on `main`, mark matching Silver criteria Met on https://www.bestpractices.dev/en/projects/5420/silver

## PR / commit disclosure

AI assistance was used to research existing evidence and draft these documentation changes for issue #7267. Human review is required before marking badge criteria Met.
