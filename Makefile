BUILD_DIR = builds
MODULE = github.com/soerenschneider/sc
BINARY_NAME = sc
CHECKSUM_FILE = $(BUILD_DIR)/checksum.sha256
SIGNIFY_KEYFILE = ~/.signify/github.sec
SIGNIFY_PASS_ENTRY = keys/signify/github

# cosign key pair to sign with. If empty, cosign signs keyless, which sc only
# trusts when done by the release workflow, so that is restricted to CI.
COSIGN_KEY ?=
# Signature schemes the checksum file is signed with, override e.g. with
# "make sign SIGNERS=cosign". Each one needs a matching sign-<signer> target.
SIGNERS ?= signify $(if $(COSIGN_KEY),cosign)
DOCKER_PREFIX = ghcr.io/soerenschneider

generate:
	go generate  ./...

install: build
	cp sc ~/bin/

tests:
	go test ./... -race -covermode=atomic -coverprofile=coverage.out
	go tool cover -html=coverage.out -o=coverage.html
	go tool cover -func=coverage.out -o=coverage.out

clean:
	git diff --quiet || { echo 'Dirty work tree' ; false; }
	rm -rf ./$(BUILD_DIR)

build: version-info generate
	CGO_ENABLED=1 go build -ldflags="-w -X '$(MODULE)/internal.BuildVersion=${VERSION}' -X '$(MODULE)/internal.CommitHash=${COMMIT_HASH}'" -o $(BINARY_NAME) .

release: clean version-info cross-build checksums

checksums: version-info
	{ echo "# version: $(VERSION)"; sha256sum $(BUILD_DIR)/sc-*; } > $(CHECKSUM_FILE)

signed-release: release sign
	gh-upload-assets -o soerenschneider -r sc -f ~/.gh-token builds

sign: $(addprefix sign-,$(SIGNERS))

# writes $(CHECKSUM_FILE).sig
sign-signify:
	pass $(SIGNIFY_PASS_ENTRY) | signify -S -s $(SIGNIFY_KEYFILE) -m $(CHECKSUM_FILE)

# writes $(CHECKSUM_FILE).sigstore.json, the passphrase of COSIGN_KEY is read
# from COSIGN_PASSWORD or prompted for
sign-cosign:
	@if [ -z "$(COSIGN_KEY)" ] && [ "$$CI" != "true" ]; then \
		echo "keyless signatures are only trusted from the release workflow, set COSIGN_KEY to sign locally"; false; \
	fi
	cosign sign-blob --yes $(if $(COSIGN_KEY),--key $(COSIGN_KEY)) --bundle $(CHECKSUM_FILE).sigstore.json $(CHECKSUM_FILE)

cross-build: cross-build-linux build-darwin

# The darwin binary needs cgo for the keychain, so it can only be built on macOS.
build-darwin: version-info
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1      go build -ldflags="-w -X '$(MODULE)/internal.BuildVersion=${VERSION}' -X '$(MODULE)/internal.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64   .

cross-build-linux: version-info
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0       go build -ldflags="-w -X '$(MODULE)/internal.BuildVersion=${VERSION}' -X '$(MODULE)/internal.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64    .
	GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -ldflags="-w -X '$(MODULE)/internal.BuildVersion=${VERSION}' -X '$(MODULE)/internal.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv6    .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0       go build -ldflags="-w -X '$(MODULE)/internal.BuildVersion=${VERSION}' -X '$(MODULE)/internal.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64  .

docker-build:
	docker build -t "$(DOCKER_PREFIX)/sc-server" .

version-info:
	$(eval VERSION := $(shell git describe --tags --abbrev=0 || echo "dev"))
	$(eval COMMIT_HASH := $(shell git rev-parse HEAD))

fmt:
	find . -iname "*.go" -exec go fmt {} \; 

pre-commit-init:
	pre-commit install
	pre-commit install --hook-type commit-msg

pre-commit-update:
	pre-commit autoupdate

.PHONY: docs
docs:
	rm -rf docs/cli/*.md
	go run -tags=docgen main.go docs
