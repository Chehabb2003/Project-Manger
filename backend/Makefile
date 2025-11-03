.PHONY: all fmt vet test fuzz build sign sbom

all: fmt vet test build

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

fuzz:
	go test -fuzz=Fuzz -fuzztime=15s ./tests

build:
	go build ./...

sbom:
	@echo "(optional) generate SBOM with syft/grype here"

sign:
	@echo "(optional) sign artifacts with cosign here"
