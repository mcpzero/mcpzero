IMAGE   ?= golang:1.22-bookworm
RUN     = docker compose run --rm golang
VERSION ?= 0.1.0-dev

.PHONY: build dist dist-macos dist-windows dist-linux dist-linux-arm64 dist-desktop dist-cp dist-checksums release release-dry test shell tidy

# Quick single-target build inside Docker (linux/arm64 on Apple Silicon hosts).
build:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) go build -ldflags "-s -w -X github.com/mcpzero/mcpzero/cli/internal/version.Version=$(VERSION)" -o dist/mcpzero ./cmd/mcpzero

# Cross-compile all supported platforms.
dist:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) ./scripts/build-all.sh

# macOS: Apple Silicon + Intel.
dist-macos:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) ./scripts/build-all.sh darwin/arm64 darwin/amd64

# Windows: x86_64.
dist-windows:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) ./scripts/build-all.sh windows/amd64

# Linux: amd64 + arm64.
dist-linux:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) ./scripts/build-all.sh linux/amd64 linux/arm64

# Linux ARM64 only (e.g. AWS Graviton, ARM VPS).
dist-linux-arm64:
	@mkdir -p dist
	$(RUN) env VERSION=$(VERSION) ./scripts/build-all.sh linux/arm64

# Release bundle: macOS + Windows + Linux ARM64.
dist-desktop: dist-macos dist-windows dist-linux

# Install the macOS arm64 build to ~/.local/bin. Use `install` (temp file +
# atomic rename) rather than `cp`: overwriting the executable in place reuses
# the same inode, and macOS keeps a per-vnode code-signature cache that goes
# stale on an in-place rewrite, making the next exec die with "Killed: 9".
# Replacing via rename gives a fresh inode and sidesteps the stale cache.
dist-cp:
	@mkdir -p ~/.local/bin
	@install -m 0755 dist/mcpzero-darwin-arm64 ~/.local/bin/mcpzero
	@mcpzero version

dist-checksums: dist
	@cd dist && shasum -a 256 mcpzero-* > SHA256SUMS && cat SHA256SUMS

# Build, package (tar.gz/zip + SHA256SUMS), and upload to R2 → https://mcpzero.io/dl/
# Cross-compiles via the build container, then packages/uploads on the host
# (needs wrangler authenticated). Usage: make release VERSION=0.1.0
release: dist
	@SKIP_BUILD=1 VERSION=$(VERSION) ./scripts/release.sh $(VERSION)

# Same as release but builds + packages locally without uploading.
release-dry: dist
	@SKIP_BUILD=1 DRY_RUN=1 VERSION=$(VERSION) ./scripts/release.sh $(VERSION)

test:
	$(RUN) go test ./...

shell:
	$(RUN) bash

tidy:
	$(RUN) go mod tidy
