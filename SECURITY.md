# Security Policy

`remote-monitor` connects to remote Linux hosts over SSH, streams a Bash sampler, and renders local terminal output. Security reports are welcome when they affect this repository, its build pipeline, or the sampler behavior it ships.

## Supported Versions

Security reports are accepted for the current `main` branch.

## Reporting a Vulnerability

Please do not open a public issue for sensitive security reports.

Report vulnerabilities privately to the project owner through GitHub. Include:

- A concise description of the issue.
- Steps to reproduce.
- Affected files, commands, workflows, or host assumptions.
- Expected impact.
- Any suggested fix or mitigation.

## Scope

In scope:

- Vulnerabilities in the Go application code.
- Unsafe sampler behavior shipped by this repository.
- SSH command construction or terminal rendering issues caused by this project.
- Dependency or build configuration issues that affect this project.

Out of scope:

- General Linux host hardening.
- Compromised SSH credentials, keys, agents, or user shell profiles.
- Vulnerabilities in third-party SSH servers, terminal emulators, or system tools unless `remote-monitor` configuration makes them exploitable.
- Attacks requiring local machine access beyond the user's existing ability to run the application.
