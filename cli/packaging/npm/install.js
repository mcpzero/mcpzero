#!/usr/bin/env node
// Downloads the prebuilt `mcpzero` binary matching this package version and the
// host platform from the mcpzero repo's GitHub Releases, and unpacks it into
// ./bin so the `mcpzero` bin shim can exec it.
"use strict";

const fs = require("fs");
const path = require("path");
const https = require("https");
const { execFileSync } = require("child_process");

// GitHub Release assets: <base>/v<version>/mcpzero_<version>_<os>_<arch>.<ext>
const BASE_URL = (process.env.MCPZERO_BASE_URL || "https://github.com/mcpzero/mcpzero/releases/download").replace(/\/$/, "");
const VERSION = require("./package.json").version;

const OS_MAP = { darwin: "darwin", linux: "linux", win32: "windows" };
const ARCH_MAP = { x64: "amd64", arm64: "arm64" };

function fail(msg) {
  console.error(`mcpzero-cli: ${msg}`);
  process.exit(1);
}

const os = OS_MAP[process.platform];
const arch = ARCH_MAP[process.arch];
if (!os || !arch) fail(`unsupported platform ${process.platform}/${process.arch}`);

const ext = os === "windows" ? "zip" : "tar.gz";
const asset = `mcpzero_${VERSION}_${os}_${arch}.${ext}`;
const url = `${BASE_URL}/v${VERSION}/${asset}`;

const binDir = path.join(__dirname, "bin");
fs.mkdirSync(binDir, { recursive: true });
const archivePath = path.join(binDir, asset);

function download(u, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error("too many redirects"));
    https
      .get(u, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          return resolve(download(res.headers.location, dest, redirects + 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${u}`));
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
        file.on("error", reject);
      })
      .on("error", reject);
  });
}

(async () => {
  try {
    await download(url, archivePath);
    if (ext === "zip") {
      execFileSync("tar", ["-xf", archivePath, "-C", binDir]); // bsdtar handles zip on macOS/Win
    } else {
      execFileSync("tar", ["-xzf", archivePath, "-C", binDir]);
    }
    fs.unlinkSync(archivePath);
    const binName = os === "windows" ? "mcpzero.exe" : "mcpzero";
    fs.chmodSync(path.join(binDir, binName), 0o755);
  } catch (err) {
    fail(`failed to install binary (${asset}): ${err.message}\n` +
      `See https://mcpzero.io/docs/cli/install/ for manual installation.`);
  }
})();
