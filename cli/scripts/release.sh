#!/usr/bin/env bash
# Build, package, and publish the mcpzero CLI to https://mcpzero.io/dl/.
#
# Artifacts are uploaded to the Cloudflare R2 bucket that the web Worker serves
# under /dl/* (see saas/web/workers/downloads.ts). Releasing the CLI therefore
# does NOT require redeploying the web Worker.
#
# Layout published per release (VERSION = 0.1.0 → tag v0.1.0):
#
#   dl/v0.1.0/mcpzero-cli_0.1.0_darwin_arm64.tar.gz   (binary named `mcpzero`)
#   dl/v0.1.0/mcpzero-cli_0.1.0_darwin_amd64.tar.gz
#   dl/v0.1.0/mcpzero-cli_0.1.0_linux_amd64.tar.gz
#   dl/v0.1.0/mcpzero-cli_0.1.0_linux_arm64.tar.gz
#   dl/v0.1.0/mcpzero-cli_0.1.0_windows_amd64.zip
#   dl/v0.1.0/SHA256SUMS
#   dl/latest/mcpzero-cli_<os>_<arch>.tar.gz          (versionless copies)
#   dl/latest/SHA256SUMS
#   dl/latest/VERSION                              (plain text, e.g. "0.1.0")
#
# The same versioned artifacts are also mirrored to the mcpzero repo's GitHub
# Releases (github.com/mcpzero/mcpzero/releases) via `gh`, unless SKIP_GH=1.
#
# Usage:
#   ./scripts/release.sh 0.1.0                 # build + package + upload (R2 + GitHub)
#   VERSION=0.1.0 ./scripts/release.sh         # same
#   SKIP_BUILD=1 ./scripts/release.sh 0.1.0    # reuse existing dist/ binaries
#   DRY_RUN=1 ./scripts/release.sh 0.1.0       # build + package, skip all uploads
#   SKIP_GH=1 ./scripts/release.sh 0.1.0       # R2 only, skip GitHub Release
#   SKIP_R2=1 ./scripts/release.sh 0.1.0       # GitHub Release only, skip R2
#   R2_BUCKET=mcpzero-downloads-dev ./scripts/release.sh 0.1.0   # target dev bucket
#
# Requirements: bash, tar, zip, shasum/sha256sum, wrangler (authenticated, for
# R2), and gh (authenticated, for the GitHub Release mirror; skip with SKIP_GH=1).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)" # cli repo root
DIST="$ROOT/dist"
STAGE="$DIST/release"

# Archive (tarball/zip) base name. The binary packed inside stays `mcpzero`.
PKG="mcpzero-cli"

SKIP_BUILD="${SKIP_BUILD:-0}"
DRY_RUN="${DRY_RUN:-0}"
SKIP_R2="${SKIP_R2:-0}"
R2_BUCKET="${R2_BUCKET:-mcpzero-downloads}"
# Sibling homebrew-tap checkout (mcpzero/homebrew-tap). Override in CI.
HOMEBREW_TAP_DIR="${HOMEBREW_TAP_DIR:-$ROOT/../homebrew-tap}"

log()  { printf '\n\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*" >&2; }
die()  { printf '\n\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

VERSION="${VERSION:-${1:-}}"
[ -n "$VERSION" ] || die "version required: ./scripts/release.sh <version>  (e.g. 0.1.0)"
VERSION="${VERSION#v}" # normalize: store without leading v
TAG="v$VERSION"

need() { command -v "$1" >/dev/null 2>&1 || die "missing command: $1"; }
need tar

if command -v shasum >/dev/null 2>&1; then
  SHACMD="shasum -a 256"
elif command -v sha256sum >/dev/null 2>&1; then
  SHACMD="sha256sum"
else
  die "need shasum or sha256sum"
fi

# wrangler runner: prefer global; else a checkout pointed to by WRANGLER_DIR
# (e.g. the saas/web workspace with wrangler installed). Only required when the
# R2 upload is actually going to run.
if [ "$SKIP_R2" = "1" ] || [ "$DRY_RUN" = "1" ]; then
  WRANGLER() { command wrangler "$@"; } # only used if R2 upload runs
elif command -v wrangler >/dev/null 2>&1; then
  WRANGLER() { wrangler "$@"; }
elif [ -n "${WRANGLER_DIR:-}" ] && [ -d "$WRANGLER_DIR/node_modules" ]; then
  WRANGLER() { (cd "$WRANGLER_DIR" && pnpm exec wrangler "$@"); }
else
  die "wrangler not found (install it globally, set WRANGLER_DIR, or pass SKIP_R2=1)"
fi

# Platforms: tarball name (os_arch) → built binary file name in dist/.
PLATFORMS="
darwin_arm64:mcpzero-darwin-arm64
darwin_amd64:mcpzero-darwin-amd64
linux_amd64:mcpzero-linux-amd64
linux_arm64:mcpzero-linux-arm64
windows_amd64:mcpzero-windows-amd64.exe
"

# ---- build -------------------------------------------------------------------
if [ "$SKIP_BUILD" = "1" ]; then
  log "Skipping build (SKIP_BUILD=1) — reusing $DIST"
else
  log "Building all platforms (VERSION=$VERSION)"
  VERSION="$VERSION" "$ROOT/scripts/build-all.sh"
fi

# ---- package -----------------------------------------------------------------
log "Packaging artifacts"
rm -rf "$STAGE"
mkdir -p "$STAGE/$TAG" "$STAGE/latest"

pkgtmp="$(mktemp -d "${TMPDIR:-/tmp}/mcpzero-pkg.XXXXXX")"
trap 'rm -rf "$pkgtmp"' EXIT

for entry in $PLATFORMS; do
  osarch="${entry%%:*}"
  binfile="${entry##*:}"
  src="$DIST/$binfile"
  [ -f "$src" ] || die "missing binary: $src (run without SKIP_BUILD, or build first)"

  if [ "${osarch%_*}" = "windows" ]; then
    need zip
    work="$pkgtmp/$osarch"
    mkdir -p "$work"
    cp "$src" "$work/mcpzero.exe"
    versioned="${PKG}_${VERSION}_${osarch}.zip"
    latest="${PKG}_${osarch}.zip"
    (cd "$work" && zip -q -X "$STAGE/$TAG/$versioned" mcpzero.exe)
  else
    work="$pkgtmp/$osarch"
    mkdir -p "$work"
    cp "$src" "$work/mcpzero"
    chmod +x "$work/mcpzero"
    versioned="${PKG}_${VERSION}_${osarch}.tar.gz"
    latest="${PKG}_${osarch}.tar.gz"
    tar -czf "$STAGE/$TAG/$versioned" -C "$work" mcpzero
  fi
  cp "$STAGE/$TAG/$versioned" "$STAGE/latest/$latest"
  ok "packaged $versioned"
done

# ---- checksums + version pointer --------------------------------------------
( cd "$STAGE/$TAG"   && $SHACMD "${PKG}"_* > SHA256SUMS )
( cd "$STAGE/latest" && $SHACMD "${PKG}"_* > SHA256SUMS )
printf '%s\n' "$VERSION" > "$STAGE/latest/VERSION"
ok "generated SHA256SUMS + VERSION"

# ---- bump Homebrew formula ---------------------------------------------------
# Keeps homebrew-tap pinned to this release (R2 URLs + matching sha256).
FORMULA="$HOMEBREW_TAP_DIR/Formula/mcpzero.rb"
sha_for() {
  grep " .${PKG}_${VERSION}_$1\.tar\.gz\$" "$STAGE/$TAG/SHA256SUMS" \
    | awk '{print $1}' | head -n1
}
if [ "${SKIP_FORMULA:-0}" = "1" ]; then
  warn "SKIP_FORMULA=1 — leaving Homebrew formula untouched"
elif [ ! -f "$FORMULA" ]; then
  warn "formula not found at $FORMULA — skipping bump"
elif ! command -v perl >/dev/null 2>&1; then
  warn "perl not available — skipping Homebrew formula bump"
elif [ "$DRY_RUN" = "1" ]; then
  printf '   would bump %s → version %s + sha256 (darwin/linux × arm64/amd64)\n' \
    "$FORMULA" "$VERSION"
else
  perl -0pi -e "s/^(  version )\"[^\"]*\"/\${1}\"$VERSION\"/m" "$FORMULA"
  for osarch in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64; do
    sha="$(sha_for "$osarch")"
    [ -n "$sha" ] || die "no checksum for $osarch in $STAGE/$TAG/SHA256SUMS"
    perl -0pi -e "s/(_${osarch}\.tar\.gz\"\s*\n\s*sha256 \")[^\"]*/\${1}$sha/" "$FORMULA"
  done
  ok "bumped $(basename "$FORMULA") → $VERSION (commit + push homebrew-tap separately)"
fi

# ---- upload ------------------------------------------------------------------
content_type() {
  case "$1" in
    *.tar.gz) echo "application/gzip" ;;
    *.zip)    echo "application/zip" ;;
    *)        echo "text/plain; charset=utf-8" ;;
  esac
}

put() {
  local file="$1" key="$2"
  local ct; ct="$(content_type "$file")"
  if [ "$DRY_RUN" = "1" ]; then
    printf '   would upload %s → r2://%s/%s (%s)\n' "${file#$STAGE/}" "$R2_BUCKET" "$key" "$ct"
    return 0
  fi
  WRANGLER r2 object put "$R2_BUCKET/$key" --file "$file" --content-type "$ct" --remote >/dev/null
  printf '   uploaded dl/%s\n' "$key"
}

if [ "$SKIP_R2" = "1" ]; then
  warn "SKIP_R2=1 — skipping Cloudflare R2 upload"
else
  if [ "$DRY_RUN" = "1" ]; then
    log "DRY_RUN=1 — packaged into $STAGE, skipping upload"
  else
    log "Checking wrangler auth"
    WRANGLER whoami >/dev/null 2>&1 || die "wrangler not authenticated — run: wrangler login (or set CLOUDFLARE_API_TOKEN)"
  fi

  log "Publishing to r2://$R2_BUCKET (served at https://mcpzero.io/dl/)"
  for f in "$STAGE/$TAG"/*; do
    put "$f" "$TAG/$(basename "$f")"
  done
  for f in "$STAGE/latest"/*; do
    put "$f" "latest/$(basename "$f")"
  done
fi

# ---- GitHub Release (mirror of the versioned artifacts) ---------------------
# Publishes the same versioned tarballs/zip + SHA256SUMS to the mcpzero repo's
# GitHub Releases, so the binaries are available both at mcpzero.io/dl (R2) and
# github.com/mcpzero/mcpzero/releases. Set GH_REPO to override the target repo.
if [ "${SKIP_GH:-0}" = "1" ]; then
  warn "SKIP_GH=1 — skipping GitHub Release"
elif ! command -v gh >/dev/null 2>&1; then
  warn "gh CLI not found — skipping GitHub Release (install: https://cli.github.com)"
elif [ "$DRY_RUN" = "1" ]; then
  printf '   would publish GitHub Release %s with %s/*\n' "$TAG" "${STAGE#"$ROOT"/}/$TAG"
else
  log "Publishing GitHub Release $TAG"
  gh auth status >/dev/null 2>&1 || die "gh not authenticated — run: gh auth login"
  gh_assets=()
  for f in "$STAGE/$TAG"/*; do gh_assets+=("$f"); done
  if gh release view "$TAG" >/dev/null 2>&1; then
    gh release upload "$TAG" "${gh_assets[@]}" --clobber
  else
    gh release create "$TAG" "${gh_assets[@]}" \
      --title "$TAG" --notes "MCPZERO CLI $TAG — install: https://mcpzero.io/install.sh"
  fi
  ok "GitHub Release $TAG published"
fi

log "Release $TAG complete"
cat <<EOF

Verify:
  curl -fsSL https://mcpzero.io/dl/latest/VERSION
  curl -fsSL https://mcpzero.io/install.sh | sh

Pinned download:
  https://mcpzero.io/dl/$TAG/mcpzero-cli_${VERSION}_darwin_arm64.tar.gz
EOF
