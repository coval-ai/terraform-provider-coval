import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, symlink, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import semanticRelease from "semantic-release";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(scriptDirectory, "../..");
const temporaryRoot = await mkdtemp(path.join(os.tmpdir(), "semantic-release-test-"));
const testRepository = path.join(temporaryRoot, "repository");
const remoteRepository = path.join(temporaryRoot, "remote.git");

function git(cwd, ...args) {
  return execFileSync("git", args, { cwd, encoding: "utf8" }).trim();
}

function testEnvironment() {
  const environment = { ...process.env };
  for (const name of Object.keys(environment)) {
    if (name === "CI" || name.startsWith("GITHUB_")) {
      delete environment[name];
    }
  }
  return environment;
}

async function releaseConfig() {
  const config = JSON.parse(await readFile(path.join(repositoryRoot, ".releaserc.json"), "utf8"));
  return {
    ...config,
    ci: false,
    repositoryUrl: pathToFileURL(remoteRepository).href,
    plugins: config.plugins.filter(([name]) => name !== "@semantic-release/github"),
  };
}

async function commitFile(name, contents, message) {
  await writeFile(path.join(testRepository, name), contents);
  git(testRepository, "add", name);
  git(testRepository, "commit", "--quiet", "--message", message);
}

try {
  await mkdir(testRepository);
  await mkdir(path.join(testRepository, "scripts", "release"), { recursive: true });
  await writeFile(path.join(testRepository, ".gitignore"), "node_modules\n");
  await writeFile(path.join(testRepository, "CHANGELOG.md"), "# Changelog\n");
  await writeFile(path.join(testRepository, "README.md"), "# Release test\n");
  await cp(
    path.join(repositoryRoot, "scripts", "release", "verify-generated-changelog.sh"),
    path.join(testRepository, "scripts", "release", "verify-generated-changelog.sh"),
  );
  await symlink(path.join(repositoryRoot, "node_modules"), path.join(testRepository, "node_modules"), "dir");

  git(temporaryRoot, "init", "--bare", "--quiet", remoteRepository);
  git(testRepository, "init", "--quiet", "--initial-branch=main");
  git(testRepository, "config", "user.name", "Release Test");
  git(testRepository, "config", "user.email", "release-test@example.com");
  git(testRepository, "config", "commit.gpgsign", "false");
  git(testRepository, "remote", "add", "origin", remoteRepository);
  git(testRepository, "add", ".gitignore", "CHANGELOG.md", "README.md", "scripts/release/verify-generated-changelog.sh");
  git(testRepository, "commit", "--quiet", "--message", "feat: add the first public resource");
  git(testRepository, "push", "--quiet", "--set-upstream", "origin", "main");

  const environment = testEnvironment();
  const result = await semanticRelease(await releaseConfig(), { cwd: testRepository, env: environment });

  assert.equal(result?.nextRelease.version, "1.0.0");
  assert.equal(git(testRepository, "show", "--no-patch", "--format=%s", "HEAD"), "chore(release): 1.0.0");
  assert.equal(git(testRepository, "tag", "--points-at", "HEAD"), "v1.0.0");
  assert.equal(git(testRepository, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"), "CHANGELOG.md");
  assert.match(await readFile(path.join(testRepository, "CHANGELOG.md"), "utf8"), /1\.0\.0/);

  await commitFile("chore-1.txt", "one\n", "chore: perform maintenance one");
  await commitFile("chore-2.txt", "two\n", "chore: perform maintenance two");
  await commitFile("chore-3.txt", "three\n", "chore: perform maintenance three");
  await commitFile("chore-4.txt", "four\n", "chore: perform maintenance four");
  await commitFile("fix.txt", "fixed\n", "fix: correct provider behavior");
  git(testRepository, "push", "--quiet", "origin", "main");

  const retryResult = await semanticRelease(await releaseConfig(), { cwd: testRepository, env: environment });
  assert.equal(retryResult?.nextRelease.version, "1.0.1");
  assert.equal(git(testRepository, "show", "--no-patch", "--format=%s", "HEAD"), "chore(release): 1.0.1");
  assert.equal(git(testRepository, "tag", "--points-at", "HEAD"), "v1.0.1");
  assert.match(await readFile(path.join(testRepository, "CHANGELOG.md"), "utf8"), /1\.0\.1/);

  const releasedHead = git(testRepository, "rev-parse", "HEAD");
  const releasedChangelog = await readFile(path.join(testRepository, "CHANGELOG.md"), "utf8");
  const noOpResult = await semanticRelease(await releaseConfig(), { cwd: testRepository, env: environment });
  assert.equal(noOpResult, false);
  assert.equal(git(testRepository, "rev-parse", "HEAD"), releasedHead);
  assert.equal(await readFile(path.join(testRepository, "CHANGELOG.md"), "utf8"), releasedChangelog);
  assert.equal(git(testRepository, "tag", "--sort=version:refname"), "v1.0.0\nv1.0.1");

  console.log("semantic release tests passed: initial, accumulated patch, and no-op retry");
} finally {
  await rm(temporaryRoot, { recursive: true, force: true });
}
