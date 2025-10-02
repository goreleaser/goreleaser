# Threat Modeling Document

## Introduction

GoReleaser is an open-source release automation tool designed to build, package,
and publish releases for multiple programming languages.

This document identifies security threats, assets, and mitigations.

## Asset Inventory

### Critical Assets

- **Source Code:** Project code, build scripts, and configuration files (e.g., `.goreleaser.yml`)
- **Build Artifacts:** Packages, binaries, containers, and other distributable outputs
- **Secrets:** API tokens, signing keys, repository credentials
- **Release Metadata:** Version numbers, changelogs
- **CI/CD Pipelines & Runners:** Automation resources executing releases
- **Third-party Dependencies:** Libraries, plugins, and integrations
- **User Data:** Data handled by project integrations

### Asset Locations

- Local developer machines
- GitHub Actions runners
- Artifact repositories
- Public package registries
- Source control platforms

## Threat Model

### Actors

- **Maintainers & Contributors:** Trusted users with varying permissions
- **External Attackers:** Untrusted users seeking to compromise releases or assets
- **Supply Chain Threats:** Malicious dependencies or compromised third-party services
- **CI/CD Systems:** Automated agents that may be exploited if misconfigured

### Entry Points

- Source code contributions (pull requests, issues)
- Configuration files and scripts
- CI/CD integration and environment variables
- Third-party plugins and dependencies
- Release pipelines and artifact repositories

### Trust Boundaries

- Between project repository and CI/CD environment
- Between GoReleaser and external plugins/dependencies
- Between artifact generation and distribution channels

### Threats

#### Supply Chain Attacks

- Compromised dependencies or plugins
- Unauthorized changes to source/configuration
- Exploitation of third-party CI/CD or repository services

#### Secrets Leakage

- Exposure of tokens, credentials, or signing keys in logs, error messages, or artifacts
- Hardcoded secrets in code or configuration
- Improper secret management in CI/CD environments

#### Code Execution/Injection

- Malicious code execution via PRs, plugins, or configuration
- Remote code execution vulnerabilities in GoReleaser or dependencies

#### Unauthorized Access

- Unauthorized users triggering releases or accessing sensitive artifacts
- Insecure permissions on runners, repositories, or artifact stores

#### Data Integrity & Tampering

- Tampering with build artifacts, changelogs, or metadata
- Compromise of signing keys, leading to malicious releases

#### Denial of Service

- Abuse of CI/CD resources, bandwidth, or artifact storage
- Overloading automated processes or API endpoints

## Mitigations

### Supply Chain Security

- Pin dependencies and use trusted sources
- Mandatory code review and CI checks on all incoming PRs
- Signed commits and release tags
- Enable immutable releases
- Run security scans on every commit

### Secrets Management

- Secure storage using environment variables and secret managers (e.g. GitHub Secrets)
- Never log or expose secrets in build or release outputs
- Regularly rotate secrets and monitor for suspicious activity

### Secure Code Execution

- Validate and sanitize configuration files and user inputs
- Limit shell command and script execution scope
- Audit dependencies and plugins for vulnerabilities

### Access Control

- Enforce least privilege for CI/CD runners, repositories, and artifact stores
- Require multi-factor authentication for maintainers
- Restrict release triggers to authorized users/systems
- Lower permissions of less active maintainers

### Artifact Integrity

- Sign release artifacts with GPG or similar tools
- Verify signatures before distribution
- Use trusted, access-controlled artifact repositories

### Availability Protection

- Implement rate limiting and resource quotas on CI/CD jobs
- Monitor for abnormal activity and automate alerts

## Residual Risks

- Zero-day vulnerabilities in dependencies, CI/CD systems, or GoReleaser itself
- Social engineering attacks targeting maintainers
- Unnoticed supply chain compromises
- Human error in configuration or secret management

## Security Best Practices

- Regularly update GoReleaser and dependencies
- Monitor security advisories and patch vulnerabilities promptly
- Educate contributors on secure coding and secrets hygiene
- Document security policies and incident response procedures

## References

- [GoReleaser Documentation](https://goreleaser.com/)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Supply Chain Security](https://slsa.dev/)
- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
