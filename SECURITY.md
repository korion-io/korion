# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Korion, please report it
privately rather than opening a public GitHub issue.

- Open a [GitHub Security Advisory](https://github.com/korion-io/korion/security/advisories/new)
  against this repository, or
- Email the maintainers (contact to be published once the org's security
  contact is set up).

Please include:
- A description of the vulnerability and its potential impact
- Steps to reproduce
- Affected version(s), if known

We will acknowledge reports within 5 business days and aim to provide a fix
or mitigation timeline within 30 days, consistent with CNCF project
expectations.

## Scope

Korion's controller holds cluster-wide read access to discover platform
topology (see RBAC in `config/rbac/`) and, once ARIA is enabled, calls an
external LLM provider with assembled platform context. Vulnerabilities
involving privilege escalation via the controller's RBAC, credential handling
for connected tools (ArgoCD, GitHub, Prometheus, etc.), or unintended data
exposure to the LLM provider are all in scope.

ARIA never executes write operations against the cluster (see `CLAUDE.md`
rule #3) — reports assuming otherwise should first confirm this invariant
still holds in the code before filing.
