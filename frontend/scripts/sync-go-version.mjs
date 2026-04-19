#!/usr/bin/env node
/**
 * Aligns repo-root Go version strings with semantic-release next version.
 * Invoked from frontend/ via @semantic-release/exec prepareCmd.
 */
import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const raw = process.argv[2]
if (!raw) {
  console.error("sync-go-version: missing version argument")
  process.exit(1)
}
const semver = raw.replace(/^v/i, "")
const vTag = `v${semver}`
const root = path.join(__dirname, "..", "..")

const edits = [
  {
    file: path.join(root, "version.go"),
    re: /(version\s*=\s*")[^"]+(")/,
    repl: `$1${vTag}$2`,
  },
  {
    file: path.join(root, "internal", "version", "version.go"),
    re: /(Current = ")[^"]+(")/,
    repl: `$1${vTag}$2`,
  },
]

for (const { file, re, repl } of edits) {
  const s = fs.readFileSync(file, "utf8")
  const next = s.replace(re, repl)
  if (next === s) {
    console.error(`sync-go-version: pattern did not match in ${file}`)
    process.exit(1)
  }
  fs.writeFileSync(file, next)
}
console.log(`sync-go-version: set launcher version to ${vTag}`)
