# Contributing

This repository is public so customers can inspect, install, and audit the Coval Terraform provider. Coval maintains the implementation and does not accept pull requests from outside the Coval organization.

For a non-sensitive bug or missing public API behavior, open an issue with the smallest reproducible example you can share publicly. Do not include API keys, credentials, customer data, private infrastructure details, or other sensitive information. Coval will triage the report and implement accepted changes internally.

Report suspected vulnerabilities privately through the repository's Security tab as described in `SECURITY.md`. Do not disclose a vulnerability in an issue or pull request.

Automated dependency updates from Dependabot are the only external pull requests accepted by the repository automation.

## Pull request titles

Coval pull request titles must use Conventional Commit format because squash-merged titles determine the provider's next semantic version and generated changelog entry.

- `feat: add a resource` introduces a feature.
- `fix(test-set): preserve ordering` fixes existing behavior and may include an optional lowercase scope.
- `feat!: replace a schema` introduces a breaking change.
- `build`, `chore`, `ci`, `docs`, `perf`, `refactor`, `revert`, `style`, and `test` are also accepted types.

Keep implementation detail in the pull request body and write the title as a concise, customer-relevant description of the change.
