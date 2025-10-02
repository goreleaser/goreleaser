# Incident Response Plan

This document outlines how the GoReleaser team responds to security incidents,
critical bugs, or operational disruptions that could affect users or the
trustworthiness of the project.

---

## 1. Scope

This plan applies to everything in the
[goreleaser/goreleaser](https://github.com/goreleaser/goreleaser) repository,
including code, releases, and GitHub workflows.

## 2. Roles & Contacts

- **Incident Lead:** By default, [@caarlos0](https://github.com/caarlos0).
- **Security Contact:** All incidents must be reported via only
  [GitHub Security Advisories][gsa].

## 3. Detection & Reporting

**All security incidents are initially considered sensitive.**

They must be reported privately and exclusively through
[GitHub Security Advisories][gsa].

Do not disclose incidents via issues, pull requests, or public channels.

## 4. Initial Response

1. **Acknowledge** the report and thank the reporter.
2. **Assess** the severity and validity. See [CIA][cia].
3. **Engage** other maintainers if needed.
4. **Contain** the issue if possible (revoke credentials, disable workflows).

## 5. Investigation & Mitigation

- **Investigate** root cause and potential impact.
- **Mitigate**:
  - Patch vulnerabilities.
  - Rotate credentials (tokens/keys) if needed.
- **Document** all findings and actions.

## 6. Resolution Timeline

Resolution or assessment will typically be provided within **30 days** of
acknowledgment.

## 7. Communication

All communication regarding security incidents must occur exclusively through
the GitHub Security Advisories page.

Once the incident is resolved, a coordinated disclosure is agreed upon,
and a fix is released, a public summary will be published.
Typically we request a CVE as well.

## 8. Post-Incident

1. **Review** the incident and response.
2. **Update** documentation or automation as needed.
3. **Publish** an advisory for significant incidents.
4. **Credit** everyone involved unless they explicitly ask to remain anonymous.

## 9. References

[SECURITY.md](./SECURITY.md)

[gsa]: https://github.com/goreleaser/goreleaser/security/advisories/new
[cia]: https://www.energy.gov/femp/operational-technology-cybersecurity-energy-systems#cia
