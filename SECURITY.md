# Security Policy

## Current Status

OIDC Bridge is a young project and should be treated as early-stage software.

Before production use, review at least:

- HTTPS termination
- session cookie settings
- secret management
- database backup strategy
- runtime key persistence
- admin password strength
- reverse proxy hardening

## Reporting a Vulnerability

If you discover a security issue, please report it privately to the maintainer before opening a public issue.

Include:

- affected version or commit
- reproduction steps
- impact assessment
- any suggested mitigation

## Known Hardening Gaps

The following areas deserve extra review in production deployments:

- signing keys are currently generated at runtime unless you add persistence
- SQLite is the default database and may not fit every deployment model
- default local cookie settings should be reviewed behind HTTPS
- downstream OIDC interoperability should be tested per client
