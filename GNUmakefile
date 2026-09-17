default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

docs-validate:
	cd tools; go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-dir .. --provider-name coval

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish,sign
	./scripts/release/verify-artifacts.sh

automation-test:
	./scripts/ci/validate-pr-title_test.sh
	./scripts/release/verify-generated-changelog_test.sh
	./scripts/release/validate-release-tag_test.sh

.PHONY: fmt lint test build install generate docs-validate release-snapshot automation-test
