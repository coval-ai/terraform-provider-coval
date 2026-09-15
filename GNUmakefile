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

.PHONY: fmt lint test build install generate docs-validate
