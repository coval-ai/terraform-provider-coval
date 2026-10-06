import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";

const pluginNames = [
  "@semantic-release/commit-analyzer",
  "@semantic-release/release-notes-generator",
  "@semantic-release/changelog",
  "@semantic-release/exec",
  "@semantic-release/git",
  "@semantic-release/github",
];
const dependencyPolicy = 'overrides:\n  conventional-changelog-writer: 9.2.1\n  "semantic-release@25.0.9>@semantic-release/npm": "-"\n';

// The removed plugin is safe only while releases use this explicit plugin contract.
export async function validateReleaseContract(root) {
  const config = JSON.parse(await readFile(path.join(root, ".releaserc.json"), "utf8"));
  const manifest = JSON.parse(await readFile(path.join(root, "package.json"), "utf8"));
  assert.deepEqual(Object.keys(config).sort(), ["branches", "plugins", "tagFormat"]);
  assert.deepEqual(config.branches, ["main"]);
  assert.equal(config.tagFormat, "v${version}");
  assert.ok(Array.isArray(config.plugins));
  assert.deepEqual(config.plugins.map((plugin) => {
    assert.ok(Array.isArray(plugin) && plugin.length === 2);
    assert.ok(plugin[1] && typeof plugin[1] === "object" && !Array.isArray(plugin[1]));
    return plugin[0];
  }), pluginNames);
  assert.equal(manifest.devDependencies["semantic-release"], "25.0.9");
  assert.equal(manifest.packageManager, "pnpm@11.28.5");
  assert.equal(manifest.scripts.release, "node scripts/release/run-release.mjs");
  assert.equal(manifest.scripts["release:dry-run"], "node scripts/release/run-release.mjs --dry-run");
  assert.equal(await readFile(path.join(root, "pnpm-workspace.yaml"), "utf8"), dependencyPolicy);
  // Inspect installed package directories, including pnpm's isolated dependency store.
  const store = await readdir(path.join(root, "node_modules", ".pnpm"));
  assert.ok(!store.some((entry) => /^(npm@|@semantic-release\+npm@|ip-address@)/.test(entry)), "Unused npm publishing graph must not be installed");
  return config;
}
