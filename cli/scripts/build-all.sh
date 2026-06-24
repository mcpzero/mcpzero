#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist"
mkdir -p "$OUT"

default_platforms=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

if [ "$#" -gt 0 ]; then
  platforms=("$@")
else
  platforms=("${default_platforms[@]}")
fi

LDFLAGS='-s -w'
VERSION="${VERSION:-dev}"

for platform in "${platforms[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  name="mcpzero-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    name="${name}.exe"
  fi
  echo "building $name (GOOS=$GOOS GOARCH=$GOARCH)"
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
    -ldflags "$LDFLAGS -X github.com/mcpzero/mcpzero/cli/internal/version.Version=${VERSION}" \
    -o "$OUT/$name" \
    "$ROOT/cmd/mcpzero"
done

echo "done: $OUT"
ls -la "$OUT"/mcpzero-* 2>/dev/null || true
