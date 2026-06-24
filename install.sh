#!/bin/sh
# MCPZERO CLI installer.
#
#   curl -fsSL https://mcpzero.io/install.sh | sh
#
# Downloads the pre-built `mcpzero` binary for your platform and installs it onto
# your PATH. By default it pulls from the project's GitHub Releases; set
# MCPZERO_BASE_URL to install from a mirror instead (e.g. the Cloudflare R2 store
# at https://mcpzero.io/dl). POSIX sh — no bash, Go, or build tools required.
#
# Environment overrides:
#   MCPZERO_VERSION      Install a specific version (e.g. 0.1.0). Default: latest.
#   MCPZERO_INSTALL_DIR  Target bin directory. Default: ~/.local/bin (falls back
#                        to /usr/local/bin with sudo when needed).
#   MCPZERO_BASE_URL     Install from this base URL (mirror / R2 layout) instead
#                        of GitHub Releases, e.g. https://mcpzero.io/dl.
#   MCPZERO_REPO         GitHub repo for releases. Default: mcpzero/mcpzero.

set -eu

# When set, install from this base URL (mirror / R2 layout); otherwise install
# from the GitHub Releases of REPO.
BASE_URL="${MCPZERO_BASE_URL:-}"
BASE_URL="${BASE_URL%/}"
BINARY="mcpzero"      # binary inside the archive + installed command name
PKG="mcpzero-cli"     # archive (tarball/zip) base name
REPO="${MCPZERO_REPO:-mcpzero/mcpzero}"

# ---- pretty output -----------------------------------------------------------
if [ -t 1 ]; then
  BOLD="$(printf '\033[1m')"; DIM="$(printf '\033[2m')"
  RED="$(printf '\033[31m')"; GREEN="$(printf '\033[32m')"
  YELLOW="$(printf '\033[33m')"; RESET="$(printf '\033[0m')"
else
  BOLD=""; DIM=""; RED=""; GREEN=""; YELLOW=""; RESET=""
fi

info()  { printf '%s\n' "${DIM}==>${RESET} $*"; }
ok()    { printf '%s\n' "${GREEN}✓${RESET} $*"; }
warn()  { printf '%s\n' "${YELLOW}!${RESET} $*" >&2; }
die()   { printf '%s\n' "${RED}error:${RESET} $*" >&2; exit 1; }

# ---- prerequisites -----------------------------------------------------------
have() { command -v "$1" >/dev/null 2>&1; }

if have curl; then
  DL="curl -fsSL"
  DL_OUT="curl -fsSL -o"
elif have wget; then
  DL="wget -qO-"
  DL_OUT="wget -qO"
else
  die "need curl or wget installed"
fi

have tar || die "need tar installed"

# ---- detect platform ---------------------------------------------------------
os="$(uname -s)"
case "$os" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  MINGW*|MSYS*|CYGWIN*)
    die "Windows is not supported by this script. Download mcpzero-windows-amd64.exe from https://github.com/${REPO}/releases" ;;
  *) die "unsupported OS: $os" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) die "unsupported architecture: $arch" ;;
esac

# Resolve the latest release version from the GitHub /releases/latest redirect
# (avoids the API and its rate limits). Prints the version without a leading v.
gh_latest_version() {
  _loc=""
  if have curl; then
    _loc="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "${GH}/latest" 2>/dev/null || true)"
  elif have wget; then
    _loc="$(wget -S --max-redirect=20 -O /dev/null "${GH}/latest" 2>&1 \
      | awk 'tolower($1) == "location:" { u = $2 } END { print u }')"
  fi
  [ -n "$_loc" ] || return 1
  _tag="${_loc##*/}"   # .../releases/tag/v1.2.3 -> v1.2.3
  _tag="${_tag%%\?*}"  # strip any ?query suffix
  [ -n "$_tag" ] || return 1
  printf '%s' "${_tag#v}"
}

# ---- resolve version + asset -------------------------------------------------
# Two sources, selected by whether MCPZERO_BASE_URL is set:
#  * GitHub Releases (default): assets at
#    https://github.com/<repo>/releases/download/v<ver>/mcpzero-cli_<ver>_<os>_<arch>.tar.gz
#    "latest" is resolved from the /releases/latest redirect.
#  * Mirror / R2 (MCPZERO_BASE_URL set): versioned builds under <base>/v<ver>/
#    and a rolling <base>/latest/ with version-less names + a latest/VERSION file.
VERSION="${MCPZERO_VERSION:-}"

if [ -n "$BASE_URL" ]; then
  if [ -n "$VERSION" ]; then
    ver="${VERSION#v}"
    CHANNEL="v${ver}"
    ASSET="${PKG}_${ver}_${OS}_${ARCH}.tar.gz"
  else
    CHANNEL="latest"
    ASSET="${PKG}_${OS}_${ARCH}.tar.gz"
    ver="$($DL "${BASE_URL}/latest/VERSION" 2>/dev/null | head -n1 || true)"
  fi
  BASE="${BASE_URL}/${CHANNEL}"
  URL="${BASE}/${ASSET}"
  SUMS_URL="${BASE}/SHA256SUMS"
else
  GH="https://github.com/${REPO}/releases"
  if [ -n "$VERSION" ]; then
    ver="${VERSION#v}"
  else
    ver="$(gh_latest_version || true)"
    [ -n "$ver" ] || die "could not determine the latest release from ${GH}/latest
     Pin a version with MCPZERO_VERSION=<x.y.z>, or see https://mcpzero.io/docs/cli/install/"
  fi
  ASSET="${PKG}_${ver}_${OS}_${ARCH}.tar.gz"
  URL="${GH}/download/v${ver}/${ASSET}"
  SUMS_URL="${GH}/download/v${ver}/SHA256SUMS"
fi

# ---- download ----------------------------------------------------------------
tmp="$(mktemp -d "${TMPDIR:-/tmp}/mcpzero.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT INT TERM

info "downloading ${BOLD}${ASSET}${RESET}${ver:+ (v${ver})}"
if ! $DL_OUT "$tmp/$ASSET" "$URL" 2>/dev/null; then
  die "download failed: $URL
     A build may not exist for ${OS}/${ARCH}. See https://mcpzero.io/docs/cli/install/"
fi

# ---- verify checksum (best effort) ------------------------------------------
if $DL_OUT "$tmp/SHA256SUMS" "$SUMS_URL" 2>/dev/null; then
  expected="$(grep " .${ASSET}\$" "$tmp/SHA256SUMS" 2>/dev/null | awk '{print $1}' | head -n1)"
  if [ -n "$expected" ]; then
    if have shasum; then actual="$(shasum -a 256 "$tmp/$ASSET" | awk '{print $1}')"
    elif have sha256sum; then actual="$(sha256sum "$tmp/$ASSET" | awk '{print $1}')"
    else actual=""; warn "no shasum/sha256sum available — skipping checksum verification"; fi
    if [ -n "$actual" ]; then
      [ "$actual" = "$expected" ] || die "checksum mismatch for ${ASSET}"
      ok "checksum verified"
    fi
  fi
else
  warn "no SHA256SUMS published — skipping checksum verification"
fi

# ---- extract -----------------------------------------------------------------
tar -xzf "$tmp/$ASSET" -C "$tmp"
bin_path="$(find "$tmp" -type f -name "$BINARY" 2>/dev/null | head -n1)"
[ -n "$bin_path" ] || die "extracted archive did not contain a '$BINARY' binary"
chmod +x "$bin_path"

# ---- choose install dir ------------------------------------------------------
DEST="${MCPZERO_INSTALL_DIR:-}"
if [ -z "$DEST" ]; then
  DEST="$HOME/.local/bin"
fi

install_binary() {
  dir="$1"
  if mkdir -p "$dir" 2>/dev/null && [ -w "$dir" ]; then
    # Atomic replace (avoids macOS stale code-signature cache on in-place rewrite).
    install -m 0755 "$bin_path" "$dir/$BINARY" 2>/dev/null \
      || { cp "$bin_path" "$dir/$BINARY.tmp" && chmod 0755 "$dir/$BINARY.tmp" && mv -f "$dir/$BINARY.tmp" "$dir/$BINARY"; }
    return 0
  fi
  return 1
}

if install_binary "$DEST"; then
  :
elif have sudo; then
  DEST="/usr/local/bin"
  warn "installing to $DEST (requires sudo)"
  sudo install -m 0755 "$bin_path" "$DEST/$BINARY" \
    || die "failed to install to $DEST"
else
  die "cannot write to install dir and sudo is unavailable — set MCPZERO_INSTALL_DIR to a writable directory"
fi

ok "installed ${BOLD}${BINARY}${RESET} → ${DEST}/${BINARY}"

# ---- PATH hint ---------------------------------------------------------------
case ":$PATH:" in
  *":$DEST:"*) ;;
  *)
    warn "$DEST is not on your PATH. Add it, e.g.:"
    printf '%s\n' "    export PATH=\"$DEST:\$PATH\""
    ;;
esac

# ---- done --------------------------------------------------------------------
printf '\n'
if "$DEST/$BINARY" version >/dev/null 2>&1; then
  ok "$("$DEST/$BINARY" version 2>/dev/null | head -n1)"
fi
cat <<EOF

${BOLD}Next steps${RESET}
  ${BINARY} login                 ${DIM}# authenticate in your browser${RESET}
  ${BINARY} tunnel start --help   ${DIM}# expose a local MCP server${RESET}

Docs: https://mcpzero.io/docs/cli/install/
EOF
