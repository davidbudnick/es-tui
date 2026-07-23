# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x     | :white_check_mark: |
| main    | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in es-tui, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email **david@budnick.ca** with:

- A description of the vulnerability
- Steps to reproduce the issue
- Any potential impact

You can expect an initial response within 72 hours. Once the issue is confirmed, a fix will be prioritized and released as soon as possible.

## Scope

The following areas are in scope for security reports:

- Command injection or arbitrary code execution
- Credential exposure (passwords, API keys, bearer tokens, TLS material)
- Path traversal in export operations
- Self-update mechanism integrity (checksum verification bypass)
- TLS configuration weaknesses that lead to credential theft

## Out of Scope

- Elasticsearch or OpenSearch server-side vulnerabilities (report those upstream)
- Denial of service via terminal input
- Issues requiring physical access to the machine
