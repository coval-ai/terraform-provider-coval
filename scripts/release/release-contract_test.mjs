import assert from "node:assert/strict";
import { cp, mkdir, mkdtemp, readFile, rm, symlink, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createRequire } from "node:module";
import { validateReleaseContract } from "./release-contract.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const reviewedConfig = await validateReleaseContract(root);
// Exercise the pinned upstream plugin loader without calling any release hook.
const require = createRequire(import.meta.url);
const loaderUrl = pathToFileURL(path.join(path.dirname(require.resolve("semantic-release")), "lib", "get-config.js"));
const { default: getConfig } = await import(loaderUrl.href);
const loaded = await getConfig({ cwd: root, env: {}, logger: { log() {}, warn() {}, error() {}, success() {} } }, reviewedConfig);
assert.deepEqual(loaded.options.plugins, reviewedConfig.plugins);
assert.ok(Object.values(loaded.plugins).every((hook) => typeof hook === "function"));
await assert.rejects(getConfig({ cwd: root, env: {}, logger: { log() {}, warn() {}, error() {}, success() {} } }, { ...reviewedConfig, plugins: null }), /@semantic-release\/npm/);
// Reproduce the upstream package.json precedence gap, without invoking hooks.
const discoveryFixture = await mkdtemp(path.join(os.tmpdir(), "release-discovery-"));
try {
  await cp(path.join(root, ".releaserc.json"), path.join(discoveryFixture, ".releaserc.json"));
  await cp(path.join(root, "pnpm-workspace.yaml"), path.join(discoveryFixture, "pnpm-workspace.yaml"));
  const discoveryManifest = JSON.parse(await readFile(path.join(root, "package.json"), "utf8"));
  discoveryManifest.release = { publish: { draftRelease: false } };
  await writeFile(path.join(discoveryFixture, "package.json"), JSON.stringify(discoveryManifest));
  await symlink(path.join(root, "node_modules"), path.join(discoveryFixture, "node_modules"), "dir");
  const discovered = await getConfig({ cwd: discoveryFixture, env: {}, logger: { log() {}, warn() {}, error() {}, success() {} } }, structuredClone(reviewedConfig));
  assert.equal(discovered.options.plugins.find(([name]) => name === "@semantic-release/github")[1].draftRelease, false);
  await assert.rejects(validateReleaseContract(discoveryFixture), /only in \.releaserc\.json/);
} finally {
  await rm(discoveryFixture, { recursive: true, force: true });
}

const fixture = await mkdtemp(path.join(os.tmpdir(), "release-contract-"));
try {
  await mkdir(path.join(fixture, "node_modules", ".pnpm"), { recursive: true });
  const original = JSON.parse(await readFile(path.join(root, ".releaserc.json"), "utf8"));
  const manifest = JSON.parse(await readFile(path.join(root, "package.json"), "utf8"));
  async function reset() {
    for (const name of [".releaserc.json", "package.json", "pnpm-workspace.yaml"]) await cp(path.join(root, name), path.join(fixture, name));
  }
  await reset();
  await validateReleaseContract(fixture);
  for (const config of [
    { branches: original.branches, tagFormat: original.tagFormat },
    { ...original, plugins: null },
    { ...original, plugins: [] },
    { ...original, plugins: [...original.plugins, ["@semantic-release/npm", {}]] },
    { ...original, extends: "external-config" },
    { ...original, publish: ["@semantic-release/npm"] },
  ]) {
    await reset();
    await writeFile(path.join(fixture, ".releaserc.json"), JSON.stringify(config));
    await assert.rejects(validateReleaseContract(fixture));
  }
  await reset();
  await rm(path.join(fixture, ".releaserc.json"));
  await assert.rejects(validateReleaseContract(fixture));
  for (const patch of [
    { devDependencies: { ...manifest.devDependencies, "semantic-release": "25.0.10" } },
    { scripts: { ...manifest.scripts, release: "semantic-release" } },
    { packageManager: "pnpm@latest" },
    { release: { publish: { draftRelease: false } } },
  ]) {
    await reset();
    await writeFile(path.join(fixture, "package.json"), JSON.stringify({ ...manifest, ...patch }));
    await assert.rejects(validateReleaseContract(fixture));
  }
  for (const filename of [".releaserc", ".releaserc.js", "release.config.cjs"]) {
    await reset();
    await writeFile(path.join(fixture, filename), "Unexpected configuration must never execute");
    await assert.rejects(validateReleaseContract(fixture));
    await rm(path.join(fixture, filename));
  }
  await reset();
  await writeFile(path.join(fixture, "pnpm-workspace.yaml"), 'overrides:\n  conventional-changelog-writer: 9.2.1\n');
  await assert.rejects(validateReleaseContract(fixture));
  await reset();
  await mkdir(path.join(fixture, "node_modules", ".pnpm", "npm@11.19.1"));
  await assert.rejects(validateReleaseContract(fixture));
  console.log("release contract tests passed: explicit plugins, fail-closed configuration, and installed graph");
} finally {
  await rm(fixture, { recursive: true, force: true });
}
