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
	cd tools && go generate ./...

.PHONY: default build install test testacc lint fmt generate
