# Terraform Provider for Coval

A Terraform provider for managing Coval configuration through the public `/v1` API with a customer API key.

## Design

The provider exposes only configuration resources supported by Coval's public `/v1` API. Every managed resource must support a complete Terraform lifecycle through that public contract; private and administrative APIs are out of scope.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.12 or later.
- [Go](https://golang.org/doc/install) 1.25 or later.

## Configuration

Set a Coval API key in the provider block or through `COVAL_API_KEY`. The provider block takes precedence.

```terraform
provider "coval" {
  api_key = var.coval_api_key
}
```

The provider uses `https://api.coval.dev/v1` by default. Development and test environments can override it with `api_base_url` or `COVAL_API_BASE_URL`.

## Building

```shell
go install
```

## Generating documentation

The registry renders provider documentation from schema descriptions, so those descriptions are customer-facing from the first commit rather than internal notes.

```shell
make generate
```

## Releasing

Pull request titles use [Conventional Commits](https://www.conventionalcommits.org/). Because the repository squash-merges pull requests with their title as the commit subject, each merge to `main` gives the release automation an unambiguous version signal: `fix` produces a patch, `feat` produces a minor, and `!` produces a major.

After a releasable change merges, the `Coval Release Automation` GitHub App determines the next version, updates `CHANGELOG.md`, commits that generated changelog directly to `main`, creates the matching `vX.Y.Z` tag, and opens a draft GitHub Release. Changes that do not affect the public release, such as `chore` or `docs`, do not create a version.

If an automatic preparation run fails, manually run the `Release Automation` workflow from `main`. It evaluates every commit since the latest release, prepares a version for any unreleased releasable commits, and safely does nothing when the release history is already current.

The protected release workflow validates that the tag belongs to a changelog-only release commit in `main` history, then uses GoReleaser to build platform archives, include the Terraform Registry protocol manifest, generate SHA-256 checksums, and sign those checksums with the provider's GPG release key. It obtains short-lived AWS credentials through GitHub OIDC and loads signing material into permission-restricted ephemeral runner files; the private key and passphrase are never stored in this repository or in GitHub secrets.

Run a credential-free snapshot of the complete package layout locally with:

```shell
make release-snapshot
```

Do not edit `CHANGELOG.md` manually or replace the assets of a published version. Merge another conventionally titled pull request and let the automation publish a new semantic version instead.

## Testing

Unit tests need no credentials:

```shell
go test ./...
```
