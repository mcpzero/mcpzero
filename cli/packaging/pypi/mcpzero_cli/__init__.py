"""MCPZERO CLI launcher.

This package does not bundle the binary. On first run it downloads the prebuilt
``mcpzero`` binary matching this package version and the host platform from the
mcpzero repo's GitHub Releases, caches it, then execs it.
"""

from __future__ import annotations

import os
import platform
import stat
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from importlib import metadata
from pathlib import Path

# GitHub Release assets: <base>/v<version>/mcpzero_<version>_<os>_<arch>.<ext>
BASE_URL = os.environ.get(
    "MCPZERO_BASE_URL", "https://github.com/mcpzero/mcpzero/releases/download"
).rstrip("/")

_OS_MAP = {"darwin": "darwin", "linux": "linux", "windows": "windows"}
_ARCH_MAP = {
    "x86_64": "amd64",
    "amd64": "amd64",
    "arm64": "arm64",
    "aarch64": "arm64",
}


def _version() -> str:
    try:
        return metadata.version("mcpzero-cli")
    except metadata.PackageNotFoundError:
        return "0.0.0"


def _platform() -> tuple[str, str]:
    os_name = _OS_MAP.get(platform.system().lower())
    arch = _ARCH_MAP.get(platform.machine().lower())
    if not os_name or not arch:
        sys.exit(f"mcpzero-cli: unsupported platform {platform.system()}/{platform.machine()}")
    return os_name, arch


def _cache_dir() -> Path:
    base = os.environ.get("XDG_CACHE_HOME") or os.path.join(Path.home(), ".cache")
    d = Path(base) / "mcpzero" / "bin"
    d.mkdir(parents=True, exist_ok=True)
    return d


def _ensure_binary() -> Path:
    os_name, arch = _platform()
    version = _version()
    bin_name = "mcpzero.exe" if os_name == "windows" else "mcpzero"
    dest = _cache_dir() / f"{version}-{os_name}-{arch}-{bin_name}"
    if dest.exists():
        return dest

    ext = "zip" if os_name == "windows" else "tar.gz"
    asset = f"mcpzero_{version}_{os_name}_{arch}.{ext}"
    url = f"{BASE_URL}/v{version}/{asset}"

    with tempfile.TemporaryDirectory() as tmp:
        archive = Path(tmp) / asset
        try:
            urllib.request.urlretrieve(url, archive)  # noqa: S310
            if ext == "zip":
                with zipfile.ZipFile(archive) as zf:
                    zf.extractall(tmp)
            else:
                with tarfile.open(archive) as tf:
                    tf.extractall(tmp)
        except Exception as err:  # noqa: BLE001
            sys.exit(
                f"mcpzero-cli: failed to download {asset}: {err}\n"
                "See https://mcpzero.io/docs/cli/install/ for manual installation."
            )
        extracted = Path(tmp) / bin_name
        os.replace(extracted, dest)
    dest.chmod(dest.stat().st_mode | stat.S_IEXEC | stat.S_IRUSR)
    return dest


def main() -> None:
    binary = _ensure_binary()
    args = [str(binary), *sys.argv[1:]]
    if os.name == "nt":
        import subprocess

        sys.exit(subprocess.call(args))
    os.execv(str(binary), args)
