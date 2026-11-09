---
title: Security assurance case
weight: 910
---

This page summarizes why GoReleaser's published
[security expectations](/resources/security/) are considered met for the
open-source CLI. It is intentionally concise and factual. A longer asset and
threat inventory lives in the repository
[THREAT_MODEL.md](https://github.com/goreleaser/goreleaser/blob/main/THREAT_MODEL.md).
Incident handling is described in
[INCIDENT_RESPONSE.md](https://github.com/goreleaser/goreleaser/blob/main/INCIDENT_RESPONSE.md).

## Threat model (summary)

GoReleaser runs with the privileges of its caller and often with access to
release credentials. Relevant threat classes include:

- **Supply chain compromise** of dependencies, Actions, or plugins used to build
  or publish releases.
- **Secrets leakage** into logs, artifacts, or committed configuration.
- **Unsafe or unexpected command execution** driven by project config, hooks, or
  build scripts.
- **Tampering** with release artifacts or metadata after build.
- **Unauthorized publish** via stolen or over-scoped tokens.

Actors include maintainers/contributors, external attackers, compromised
third-party services, and misconfigured CI. Entry points include pull requests,
configuration files, CI environment variables, and registry/SCM APIs.

## Trust boundaries

| Boundary | Trust assumption |
| --- | --- |
| Developer workstation or CI runner ↔ GoReleaser process | The process can read the repo, env, and configured secrets; GoReleaser is not a sandbox |
| GoReleaser ↔ language toolchains / Docker / packagers | External tools are invoked as configured; their output is trusted to the degree the pipeline trusts those tools |
| GoReleaser ↔ SCM / package registries / blob stores | Authenticated API calls using caller-supplied credentials over HTTPS |
| GoReleaser project releases ↔ downstream users | Users should verify checksums/signatures before trusting binaries |
| Contributors ↔ maintainers | Changes land through GitHub review and CI; maintainers control releases |

Anything on the “untrusted config / untrusted script” side of the runner
boundary can still harm the pipeline; see
[security expectations](/resources/security/).

## Secure design principles applied

- **Least privilege in project CI:** workflows default to narrow
  `permissions` (typically `contents: read`), with write/`id-token` only where
  publishing or signing requires it
  (see `.github/workflows/*.yml`).
- **Separate credentials from code:** tokens and keys are expected via
  environment / CI secrets, not compiled into the binary
  (`crypto_credential_agility`).
- **Integrity for first-party releases:** release checksums are signed with
  Sigstore/cosign; install docs explain verification; the release workflow
  includes a `verify` job
  ([install](/getting-started/install/), `.github/workflows/release.yml`).
- **Defense in depth on contributions:** tests, `golangci-lint`, CodeQL,
  Semgrep, govulncheck, dependency review, Grype/SBOM scans, gitleaks, and
  OpenSSF Scorecard run in GitHub Actions.
- **Fail closed on pipeline errors:** a failed pipe aborts later publish/announce
  steps ([how it works](/getting-started/how-it-works/)).
- **Input handling:** configuration and CLI flags are parsed into typed
  structures and validated before work proceeds; invalid config is rejected
  rather than silently ignored where constraints exist.
- **Modern transport defaults:** outbound integrations use HTTPS through Go's
  standard TLS stack (certificate verification enabled by default).

## Common implementation weaknesses countered

| Weakness class | Countermeasures in practice |
| --- | --- |
| Injection / unexpected exec | Typed config; explicit hook/script configuration; users must treat build scripts as trusted code |
| Secrets in artifacts or VCS | gitleaks in CI; security advisories and fixes when leakage bugs are found; documented expectation to use env-based secrets |
| Vulnerable dependencies | `go.mod`/`go.sum`, Dependabot, govulncheck, dependency-review, Grype, SBOM scan |
| Weak or ad-hoc release crypto | Sigstore/cosign for checksum bundles; SHA-256 checksums; optional attestations documented for users |
| Memory-unsafety in the product language | Primary implementation is Go (memory-safe); fuzz helpers exist for sensitive parsing paths (`task fuzz`) |
| Over-privileged automation | Per-workflow GitHub token permissions; OIDC/`id-token` used for signing rather than long-lived keys where applicable |
| Silent quality regressions | Required-style CI via tests + lint; contributing policy requires tests for major new functionality |

## Residual risk

No assurance case can prove absence of defects. Residual risk includes
zero-days in dependencies or runners, social engineering of maintainers,
misconfigured user CI secrets, and malicious project-level build scripts that
GoReleaser intentionally executes as part of release automation.
