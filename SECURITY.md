# Security Policy

## Security expectations

GoReleaser is a release-automation CLI: it builds, packages, signs, and
publishes software releases based on project configuration and the environment
it runs in (typically a developer's machine or CI).

**What you can expect**

- Private vulnerability reporting and a documented response timeline (below).
- Public credit for reporters unless anonymity is requested.
- Signed release checksums (and related verification material) for GoReleaser's
  own releases, with
  [verification instructions](https://goreleaser.com/getting-started/install/).
- Ongoing dependency and security scanning in CI (for example govulncheck,
  dependency review, Grype/SBOM scans, CodeQL/Semgrep, and secret scanning).

**What this does not claim**

- GoReleaser is **not** a sandbox for untrusted build scripts, hooks, or
  toolchain plugins. A malicious or careless `.goreleaser` configuration, build
  script, or CI job can execute arbitrary commands with the privileges of that
  environment.
- Overall release security also depends on **your** CI secrets, token scopes,
  runner hardening, and registry/SCM permissions. Misconfigured tokens or
  overly broad permissions are outside what GoReleaser alone can guarantee.

For a longer write-up of threats, trust boundaries, and mitigations, see the
[assurance case](https://goreleaser.com/resources/assurance-case/) and
[THREAT_MODEL.md](./THREAT_MODEL.md).

## Supported Versions

Only the last stable version at any given point.

## Response Timeline

We aim to acknowledge vulnerability reports within 3 business days.
Resolution or assessment is typically provided within 7 business days from the
reporting date.

## Scope

We address vulnerabilities that could compromise the confidentiality, integrity,
or availability of GoReleaser or its users.

## Credit

We are happy to publicly acknowledge reporters in release notes, unless
anonymity is requested.

## Reporting a Vulnerability

Vulnerabilities can be disclosed in private using
[GitHub advisories](https://github.com/goreleaser/goreleaser/security/advisories/new).

For issues specific to GoReleaser Pro, please refer to
[this instead](https://github.com/goreleaser/goreleaser-pro/security/policy).
