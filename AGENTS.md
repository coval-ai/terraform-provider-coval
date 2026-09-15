# AGENTS.md

Rules for any AI agent working in this repository, written to `AGENTS.meta.md`. Precedence follows the [AGENTS.md standard](https://agents.md/): the closest `AGENTS.md` wins within the tree, and an explicit instruction from the user overrides this file.

How to read a rule: every rule is binding within that precedence. A rule starting `Never` means stop and name the rule to the user if the task appears to require it. A rule starting `Ask first` means stop and ask before proceeding. Any other rule is an imperative and is followed without being asked.

## Before you start

- Read `README.md` for the supported provider surface, development commands, and release contract.
- Read `AGENTS.meta.md` and follow its rules before editing `AGENTS.md` or `AGENTS.meta.md`.
- Treat every tracked file, commit, pull-request description, review comment, generated artifact, and Git history entry as public before the repository visibility changes.
- Never add credentials or sensitive data, including API keys, tokens, passwords, private keys, signing material, customer data, real account identifiers, resource ARNs, private hostnames, or secret-manager paths. Use placeholders in examples and private repository settings for operational identifiers. No mechanical check enforces this rule yet.
- Never copy non-public Coval implementation details into this repository, including private design-document content, administrative endpoints, non-public API fields, service topology, datastore schemas, incident details, or private repository paths. No mechanical check enforces this rule yet.
- Treat pull requests from outside the Coval organization as reports rather than merge candidates, and direct the author to `CONTRIBUTING.md`.
- Stop and tell the user when a requested change cannot be implemented without violating a public-repository boundary.

## Commands

- Run `make fmt` before pushing.
- Run `make generate` and review the resulting diff before pushing.
- Run `make docs-validate` before pushing.
- Run `go test -race ./...` before pushing.
- Run `go vet ./...` before pushing.
- Run `golangci-lint run` before pushing.
- Run `goreleaser check` after changing `.goreleaser.yml` or `.github/workflows/release.yml`.

## Public API

- Implement only operations and fields present in Coval's public `/v1` API contract. Stop and ask when the public contract cannot support a Terraform lifecycle instead of copying a private endpoint or backend type.
- Keep provider schema descriptions customer-facing; the Terraform Registry renders them as public documentation.
- Preserve Resource Identity and import behavior for every managed resource.
- Regenerate `docs/` with `make generate` whenever a provider, resource, or data-source schema changes.
- Use placeholders rather than real tenant IDs, object IDs, API keys, or private URLs in tests, fixtures, documentation, and examples.

## GitHub Actions

- Pin every third-party GitHub Action to a full commit SHA and retain the version comment.

## Releases

- Pin every release tool to an exact version; never use `latest`, a major-only version, or a version range in `.github/workflows/release.yml`. No mechanical check enforces this rule yet.
- Ask first before renaming the `Release` workflow or changing its `v*` tag trigger. External OIDC trust depends on both values.
- Keep the AWS account ID, role ARN, and secret identifiers in masked GitHub repository secrets, and keep the AWS region and expected public signing-key fingerprint in private repository variables rather than committed workflow values.
- Keep AWS session credentials scoped to the secret-fetching release step instead of exporting them to the job environment.
- Never store the release-signing private key or passphrase in GitHub secrets, repository files, workflow outputs, environment files, logs, caches, or artifacts. The release workflow retrieves them through short-lived OIDC credentials into permission-restricted ephemeral files. No mechanical check enforces this rule yet.
- Never echo release credentials or enable shell tracing in a step that can access them. No mechanical check enforces this rule yet.
- Remove ephemeral signing files on every release-step exit.
- Require every release tag to be a complete semantic version on the current `main` commit, and verify the imported key against the configured signing-key fingerprint before signing.
- Never create or push a version tag, publish a GitHub release, replace published assets, register the provider, or change repository visibility unless the user explicitly requests that action. No mechanical check enforces this rule yet.
- Add a changelog entry before publishing a new semantic version.

## Tests

- Keep unit tests credential-free and runnable against local HTTP test servers.
- Add proportionate unit and acceptance coverage with every managed resource or data source.
- Ask first before introducing or running acceptance tests because they create and delete live objects.
