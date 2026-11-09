# Governance (draft)

> **Draft — needs maintainer confirmation.**
> This document records the governance model implied by public project
> materials (especially the [maintainers](https://goreleaser.com/resources/maintainers/)
> page). It is not a substitute for a maintainer-signed policy until reviewed.

## Decision model

GoReleaser uses a **Benevolent Dictator For Life (BDFL)** governance model.

- Day-to-day technical direction, release decisions, and final calls on
  disputed changes rest with the project lead (BDFL).
- Contributors propose changes via GitHub pull requests, issues, and
  discussions.
- Maintainers and area owners may review, request changes, and merge within
  their areas of responsibility when they have repository permissions to do so.
- On disagreement, the BDFL decision is authoritative for this project.

This matches the public description of Carlos Alexandro Becker as
“Creator and BDFL of GoReleaser” on the maintainers page.

## Roles

Public role descriptions and named people are listed at
https://goreleaser.com/resources/maintainers/.

At a high level:

| Role | Responsibilities |
| --- | --- |
| Project lead (BDFL) | Product direction, final decision-making, releases for this repository, security incident lead by default ([INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md)) |
| Area maintainers / creators | Own specific integrations or companion projects called out on the maintainers page (for example GitHub Action, Azure DevOps extension, GoReleaser Cross, design) |
| Contributors | Propose patches, docs, and discussion; follow [CONTRIBUTING.md](./CONTRIBUTING.md) and the [code of conduct](https://github.com/goreleaser/.github/blob/main/CODE_OF_CONDUCT.md) |
| Emeritus | Previously active contributors; no current duty implied |

## What this draft does **not** claim

- It does **not** invent who holds which org secrets, DNS registrations, or
  package-registry owners beyond what maintainers publish themselves.
- It does **not** by itself satisfy OpenSSF `access_continuity` without a
  separate, maintainer-confirmed succession / access statement.

## Related documents

- [Maintainers](https://goreleaser.com/resources/maintainers/)
- [Contributing](./CONTRIBUTING.md)
- [Security policy](./SECURITY.md)
- [Roadmap (draft)](./ROADMAP.md)
- [OpenSSF Silver evidence matrix](./docs/openssf-silver.md)
