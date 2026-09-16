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

  const result = await semanticRelease(
    {
      branches: ["main"],
      ci: false,
      repositoryUrl: pathToFileURL(remoteRepository).href,
      tagFormat: "v${version}",
      plugins: [
        ["@semantic-release/commit-analyzer", { preset: "conventionalcommits" }],
        ["@semantic-release/release-notes-generator", { preset: "conventionalcommits" }],
        ["@semantic-release/changelog", { changelogFile: "CHANGELOG.md", changelogTitle: "# Changelog" }],
        ["@semantic-release/exec", { prepareCmd: "./scripts/release/verify-generated-changelog.sh ${nextRelease.version}" }],
        [
          "@semantic-release/git",
          {
            assets: ["CHANGELOG.md"],
            message: "chore(release): ${nextRelease.version}\n\n${nextRelease.notes}",
          },
        ],
      ],
    },
    { cwd: testRepository, env: process.env },
  );

  assert.equal(result?.nextRelease.version, "1.0.0");
  assert.equal(git(testRepository, "show", "--no-patch", "--format=%s", "HEAD"), "chore(release): 1.0.0");
  assert.equal(git(testRepository, "tag", "--points-at", "HEAD"), "v1.0.0");
  assert.equal(git(testRepository, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"), "CHANGELOG.md");
  assert.match(await readFile(path.join(testRepository, "CHANGELOG.md"), "utf8"), /1\.0\.0/);

  console.log("semantic release test passed: first release is v1.0.0");
} finally {
  await rm(temporaryRoot, { recursive: true, force: true });
}
