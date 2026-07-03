default: build

build:
	go build ./...

install:
	go install .

test:
	go test ./... -timeout=120s

testacc:
	TF_ACC=1 go test ./... -v -timeout=120m

lint:
	golangci-lint run

fmt:
	gofmt -s -w -e .

generate:
	cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name iru

# Build a Terraform filesystem-mirror zip for internal distribution via Iru.
# Override version with: make mirror-zip VERSION=0.2.0
VERSION ?= 0.1.0
mirror-zip:
	./scripts/build-mirror-zip.sh $(VERSION)

.PHONY: default build install test testacc lint fmt generate mirror-zip
