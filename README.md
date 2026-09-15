# Terraform Provider for Coval

A Terraform provider for managing Coval configuration through the public `/v1` API with a customer API key.

The provider can manage test sets and their test cases, look up either resource
by ID, and list the resources visible to the configured API key. Additional
resources and data sources will be added incrementally from the public API
contract.

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

Test cases belong to test sets, so configurations normally declare the two
resources together:

```terraform
resource "coval_test_set" "regression" {
  display_name = "Regression"
}

resource "coval_test_case" "refund_policy" {
  test_set_id = coval_test_set.regression.id
  input_str   = "Ask whether an item purchased three weeks ago can be returned."

  expected_behaviors = [
    "Explain the return policy",
    "Offer to begin the return process",
  ]
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

Releases are built by GitHub Actions from semantic-version tags such as
`v0.1.0`. The release workflow uses GoReleaser to build platform archives,
include the Terraform Registry protocol manifest, generate SHA-256 checksums,
and sign those checksums with the provider's GPG release key.

The release workflow obtains short-lived AWS credentials through GitHub OIDC and loads the GPG private key and passphrase into ephemeral runner files. Signing material is never stored in this repository or in GitHub secrets.

Do not replace the assets of a published version. Merge a changelog entry and
publish a new semantic version instead.

## Testing

Unit tests need no credentials:

```shell
go test ./...
```
