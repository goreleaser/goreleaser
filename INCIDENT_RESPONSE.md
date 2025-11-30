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
- **Security Contact:** All incidents must be reported exclusively through
  [GitHub Security Advisories][gsa].

## 3. Detection & Reporting

**All security incidents are initially considered sensitive** and must be
reported privately and exclusively through [GitHub Security Advisories][gsa].

Do not disclose incidents through issues, pull requests, or public channels.

## 4. Initial Response

1. **Acknowledge** the report and thank the reporter.
2. **Assess** the severity and validity (confidentiality, integrity, availability).
   See [CIA triad][cia].
3. **Engage** other maintainers if needed.
4. **Contain** the threat immediately if possible (e.g., revoke credentials,
   disable workflows).
5. **Notify Pro customers** through [Gumroad](https://gumroad.com/emails/new) if
   the incident is severe or directly affects them.

## 5. Investigation & Mitigation

- **Investigate** the root cause and potential impact.
- **Mitigate**:
  - Patch vulnerabilities.
  - Rotate compromised credentials (tokens/keys).
- **Document** all findings and actions taken.

## 6. Resolution Timeline

Resolution or assessment will typically be provided within **30 days** of
acknowledgment.

## 7. Communication

All communication regarding security incidents must occur exclusively through
the GitHub Security Advisories page.

Once the incident is resolved and a fix is released, we will:

1. Coordinate disclosure timing with the reporter.
2. Publish a public advisory summarizing the incident.
3. Request a CVE identifier if applicable.
4. Send a follow-up to Pro customers through [Gumroad](https://gumroad.com/emails/new)
   with the full resolution details (if not already notified during initial response).

## 8. Post-Incident

1. **Review** the incident response and identify lessons learned.
2. **Update** documentation, processes, or automation as needed.
3. **Publish** a public advisory for significant incidents.
4. **Credit** all contributors unless they explicitly request to remain anonymous.

## 9. References

[SECURITY.md](./SECURITY.md)

[gsa]: https://github.com/goreleaser/goreleaser/security/advisories/new
[cia]: https://www.energy.gov/femp/operational-technology-cybersecurity-energy-systems#cia
