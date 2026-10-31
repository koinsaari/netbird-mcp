# Security Policy

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Use GitHub's [Private Vulnerability Reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/report-privately) instead:

1. Go to the [Security tab](../../security) of this repository.
2. Click **"Report a vulnerability"**.
3. Fill in the form — include reproduction steps and any proposed fix.

You will be notified through GitHub when the report is acknowledged. Expect a first response within **7 days**.

## Supported versions

Only the latest release is supported.

## Scope

In scope:
- Token leakage: the NetBird personal access token appearing in logs, errors, or tool output.
- Tools that perform writes beyond what the caller requested.
- Injection into NetBird API paths or queries through tool arguments.
- Authentication bypass of the MCP transport.

Out of scope:
- Findings that require pre-existing access to the host running the server or to its environment variables.
- Vulnerabilities in NetBird itself (report those to the NetBird project).
- Misconfigurations in deployments that are not this repository's defaults.
