#!/usr/bin/env bash
# Publish the CLI distribution wrappers and reserved-name stubs.
#
#   mcpzero-cli   npm + PyPI  — wrapper packages; `mcpzero` command downloads the
#                               prebuilt binary (must match a released VERSION).
#   mcpzero       npm + PyPI  — reserved-name placeholder stubs (version 0.0.0).
#
# Run AFTER scripts/release.sh has published the matching binary to the mcpzero
# repo's GitHub Releases (github.com/mcpzero/mcpzero/releases) — the wrappers
# download that binary at install/run time.
#
# Usage:
#   ./scripts/publish-packages.sh 0.1.1            # publish wrappers at VERSION
#   STUBS=1 ./scripts/publish-packages.sh          # (re)publish reserved stubs only
#   DRY_RUN=1 ./scripts/publish-packages.sh 0.1.1  # pack/build but do not publish
#   SKIP_PYPI=1 ./scripts/publish-packages.sh 0.1.1   # npm only
#   SKIP_NPM=1 ./scripts/publish-packages.sh 0.1.1    # PyPI only
#
# Requirements: npm (logged in / NODE_AUTH_TOKEN), python3 + build + twine (for PyPI).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG="$ROOT/packaging"
DRY_RUN="${DRY_RUN:-0}"
STUBS="${STUBS:-0}"
SKIP_NPM="${SKIP_NPM:-0}"
SKIP_PYPI="${SKIP_PYPI:-0}"

log() { printf '\n\033[1;36m==>\033[0m %s\n' "$*"; }
die() { printf '\n\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

npm_publish() { # dir
  if [ "$DRY_RUN" = "1" ]; then (cd "$1" && npm pack --dry-run); else (cd "$1" && npm publish --access public); fi
}

pypi_publish() { # dir
  ( cd "$1"
    rm -rf dist
    python3 -m build
    if [ "$DRY_RUN" = "1" ]; then twine check dist/*; else twine upload dist/*; fi
  )
}

if [ "$STUBS" != "1" ]; then
  VERSION="${1:-}"
  [ -n "$VERSION" ] || die "version required (e.g. 0.1.1) — or run with STUBS=1 for stubs only"
  VERSION="${VERSION#v}"

  log "Setting wrapper versions to $VERSION"
  (cd "$PKG/npm" && npm version "$VERSION" --no-git-tag-version --allow-same-version >/dev/null)
  perl -0pi -e "s/^version = \"[^\"]*\"/version = \"$VERSION\"/m" "$PKG/pypi/pyproject.toml"

  if [ "$SKIP_NPM" = "1" ]; then log "SKIP_NPM=1 — skipping npm"; else
    log "Publishing mcpzero-cli (npm)"
    npm_publish "$PKG/npm"
  fi
  if [ "$SKIP_PYPI" = "1" ]; then log "SKIP_PYPI=1 — skipping PyPI"; else
    log "Publishing mcpzero-cli (PyPI)"
    pypi_publish "$PKG/pypi"
  fi
fi

if [ "$STUBS" = "1" ] || [ "${PUBLISH_STUBS:-0}" = "1" ]; then
  if [ "$SKIP_NPM" = "1" ]; then log "SKIP_NPM=1 — skipping npm stub"; else
    log "Publishing reserved 'mcpzero' stub (npm)"
    npm_publish "$PKG/stubs/npm"
  fi
  if [ "$SKIP_PYPI" = "1" ]; then log "SKIP_PYPI=1 — skipping PyPI stub"; else
    log "Publishing reserved 'mcpzero' stub (PyPI)"
    pypi_publish "$PKG/stubs/pypi"
  fi
fi

log "Done"
