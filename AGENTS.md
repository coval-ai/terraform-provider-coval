# AGENTS.md

Repository-specific layout, commands, integration contracts, and operational constraints. Authoring guidance for this file is in `AGENTS.meta.md`.

## Before you start

- Read `README.md` for the supported provider surface, development commands, and release contract.
- Read `AGENTS.meta.md` and follow its rules before editing `AGENTS.md` or `AGENTS.meta.md`.
- Treat every tracked file, commit, pull-request description, review comment, generated artifact, and Git history entry as public before the repository visibility changes.
- Never add credentials or sensitive data, including API keys, tokens, passwords, private keys, signing material, customer data, real account identifiers, resource ARNs, private hostnames, or secret-manager paths. Use placeholders in examples and private repository settings for operational identifiers.
- Never copy non-public Coval implementation details into this repository, including private design-document content, administrative endpoints, non-public API fields, service topology, datastore schemas, incident details, or private repository paths.
- Treat pull requests from outside the Coval organization as reports rather than merge candidates, and direct the author to `CONTRIBUTING.md`.

## Commands

- `make fmt` is the repository’s Go formatting command.
- `make generate` regenerates Terraform Registry documentation from provider schemas.
- `make docs-validate` is the repository’s generated-documentation check.
- `go test -race ./...` is the repository’s Go test command with race detection.
- `go vet ./...` is the repository’s Go vet command.
- `golangci-lint run` is the repository’s configured Go lint command.
- Run `goreleaser check` after changing `.goreleaser.yml` or `.github/workflows/release.yml`.
- Run `pnpm install --frozen-lockfile --ignore-scripts` and `pnpm test` after changing release automation configuration or scripts.
- Run `make release-snapshot` after changing release packaging.

## Public API

- Implement only operations and fields present in Coval's public `/v1` API contract. Private endpoints and backend types are outside the provider’s supported surface.
- Keep provider schema descriptions customer-facing; the Terraform Registry renders them as public documentation.
- Preserve Resource Identity and import behavior for every managed resource.
- Regenerate `docs/` with `make generate` whenever a provider, resource, or data-source schema changes.
- Use placeholders rather than real tenant IDs, object IDs, API keys, or private URLs in tests, fixtures, documentation, and examples.

## Releases

- Pin every release tool to an exact version; never use `latest`, a major-only version, or a version range in `.github/workflows/release.yml`.
- External OIDC trust binds the `Release` workflow name and its `v*` tag trigger; changing either requires a matching trust update.
- Use Conventional Commit format for pull request titles because the squash-merged title drives automated versioning and changelog generation.
- Let release automation update `CHANGELOG.md`; do not edit it in feature pull requests.
- Keep the AWS account ID, role ARN, and secret identifiers in masked GitHub repository secrets, and keep the AWS region and expected public signing-key fingerprint in private repository variables rather than committed workflow values.
- Keep AWS session credentials scoped to the secret-fetching release step instead of exporting them to the job environment.
- Never store the release-signing private key or passphrase in GitHub secrets, repository files, workflow outputs, environment files, logs, caches, or artifacts. The release workflow retrieves them through short-lived OIDC credentials into permission-restricted ephemeral files.
- Never echo release credentials or enable shell tracing in a step that can access them.
- Remove ephemeral signing files on every release-step exit.
- Require every release tag to be a complete semantic version created by release automation for a changelog-only commit in `main` history, and verify the imported key against the configured signing-key fingerprint before signing.
- Require a generated changelog entry before publishing a new semantic version.

## Tests

- Keep unit tests credential-free and runnable against local HTTP test servers.
- Acceptance tests create and delete live objects; they are not part of the credential-free local unit-test lane.
