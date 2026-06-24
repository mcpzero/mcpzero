#!/usr/bin/env node
// Thin launcher: exec the platform `mcpzero` binary unpacked by install.js.
"use strict";

const path = require("path");
const fs = require("fs");
const { spawnSync } = require("child_process");

const binName = process.platform === "win32" ? "mcpzero.exe" : "mcpzero";
const binPath = path.join(__dirname, binName);

if (!fs.existsSync(binPath)) {
  console.error(
    "mcpzero-cli: binary not found. Reinstall the package (npm install mcpzero-cli) " +
      "or install manually from https://mcpzero.io/docs/cli/install/."
  );
  process.exit(1);
}

const res = spawnSync(binPath, process.argv.slice(2), { stdio: "inherit" });
process.exit(res.status === null ? 1 : res.status);
