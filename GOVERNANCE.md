# Governance

`remote-monitor` is a public open-source project. This policy explains how maintainers guide project direction, triage issues, and review pull requests.

## Maintainers And Decisions

`remote-monitor` is maintained by the repository owner and any collaborators with maintainer access. Maintainers are responsible for repository settings, issue triage, pull request review, release direction, and final merge decisions.

Decisions are made in the open when practical, using issues and pull requests as the working record. Maintainers prioritize changes that keep the tool reliable, portable across ordinary Linux hosts, secure, maintainable, and aligned with its SSH-based terminal monitoring scope. Larger feature, policy, or architecture changes should start as an issue before implementation so maintainers can confirm the direction and scope.

Maintainers may decline or defer work that is out of scope, too broad, insufficiently specified, duplicative, unsafe, unmaintainable, or inconsistent with the license and project goals. When that happens, maintainers should leave a short explanation and use the appropriate issue state or labels.

## Issue Triage

Issues should use the bug report or feature request templates when they apply. Maintainers triage issues before work starts so the backlog stays executable:

- New or unclear issues use `status:needs-triage`.
- Ready issues use `status:ready` once they have a clear type label, area label, and next action or acceptance criteria.
- Blocked issues use `status:blocked` when another issue, maintainer decision, external service, or repository setting must be resolved first.
- `priority:high` is reserved for work that fixes a high-risk defect, protects security, or unblocks release confidence.
- Maintainers may close issues that are duplicates, completed, stale after follow-up, not planned, outside the project scope, or no longer accurate after newer work.

## Pull Request Review

Pull requests should follow the template, include relevant labels, and keep generated binaries or machine-local files out of commits. Maintainers review pull requests after the relevant local checks and required GitHub checks pass.

Maintainer review focuses on correctness, security, portability, maintainability, terminal behavior, test coverage, and fit with the SSH-based monitoring scope. Pull requests may be held, requested for changes, or closed when they do not meet those expectations.

Review is best effort. There is no guaranteed response time, but a polite follow-up is reasonable if a ready pull request has not received maintainer attention after about two weeks.

## Contribution Scope

External contributions are welcome when they fit the project's scope, especially documentation, tests, portability improvements, security fixes, bug fixes, terminal rendering polish, metrics coverage, and repository tooling.

Contributions are accepted under the existing [MIT License](LICENSE).
