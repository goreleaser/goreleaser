# Roadmap (draft)

> **Draft — needs maintainer confirmation.**
> Items below are drawn only from **public** signals (discussions, milestones,
> and ongoing project practice). Anything without a firm public commitment is
> marked as a **proposal**.

Public site mirror: https://goreleaser.com/resources/roadmap/

## Next ~12 months (working view)

### Continue the v2 line

- Ship regular v2.x releases with fixes, deprecations, and incremental
  features, following SemVer and the
  [deprecations](https://goreleaser.com/resources/deprecations/) process.
- Keep CI, dependency scanning, and docs aligned with current behavior.

### GoReleaser v3 (proposal — no public ship date)

Public discussion:
[v3: the breaking changes (#6491)](https://github.com/goreleaser/goreleaser/discussions/6491).

Milestone: [v3.0.0](https://github.com/goreleaser/goreleaser/milestone/33)
(open; no due date at the time of writing).

**Proposals discussed for v3** (not a promised checklist):

- Remove legacy `docker` / `docker_manifest` in favor of newer Docker support
- Standardize target naming (for example `-` instead of `_`)
- Remove Homebrew formula generation paths called out in the discussion
- Make the CLI default to help instead of implying `release`
- Revisit archive file defaults and the builders API
- Possibly rename snapshot/`--local` semantics
- Remove currently deprecated options

### Explicitly out of scope (unless maintainers say otherwise)

- Turning GoReleaser into a general-purpose CI system or remote build farm
- Guaranteeing sandboxing of untrusted project build scripts (see
  [security expectations](https://goreleaser.com/resources/security/))

### OpenSSF Best Practices Silver (in progress)

Track documentation and evidence work in
[issue #7267](https://github.com/goreleaser/goreleaser/issues/7267) and
[docs/openssf-silver.md](./docs/openssf-silver.md).

## Feedback

Suggestions belong in
[GitHub Discussions](https://github.com/goreleaser/goreleaser/discussions) or
issues. This file should be updated when maintainers adopt, defer, or reject
proposals.
