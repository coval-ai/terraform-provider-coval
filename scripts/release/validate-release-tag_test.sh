#!/usr/bin/env bash

set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
validator="${script_directory}/validate-release-tag.sh"
temporary_directory="$(mktemp -d)"
trap 'rm -rf -- "${temporary_directory}"' EXIT

assert_fails() {
  if "$@" >/dev/null 2>&1; then
    echo "expected command to fail: $*" >&2
    exit 1
  fi
}

repository="${temporary_directory}/repository"
remote_repository="${temporary_directory}/remote.git"
git init --bare --quiet "${remote_repository}"
git init --quiet --initial-branch=main "${repository}"
git -C "${repository}" config user.name 'Release Test'
git -C "${repository}" config user.email 'release-test@example.com'
git -C "${repository}" config commit.gpgsign false
git -C "${repository}" remote add origin "${remote_repository}"
printf '# Changelog\n' > "${repository}/CHANGELOG.md"
git -C "${repository}" add CHANGELOG.md
git -C "${repository}" commit --quiet --message 'feat: initial feature'
printf '# Changelog\n\n## 1.0.0\n' > "${repository}/CHANGELOG.md"
git -C "${repository}" add CHANGELOG.md
git -C "${repository}" commit --quiet --message 'chore(release): 1.0.0'
release_sha="$(git -C "${repository}" rev-parse HEAD)"
git -C "${repository}" push --quiet --set-upstream origin main

env REPOSITORY_ROOT="${repository}" GITHUB_REF_NAME='v1.0.0' GITHUB_ACTOR='coval-release-automation[bot]' GITHUB_SHA="${release_sha}" "${validator}" >/dev/null
assert_fails env REPOSITORY_ROOT="${repository}" GITHUB_REF_NAME='1.0.0' GITHUB_ACTOR='coval-release-automation[bot]' GITHUB_SHA="${release_sha}" "${validator}"
assert_fails env REPOSITORY_ROOT="${repository}" GITHUB_REF_NAME='v1.0.0' GITHUB_ACTOR='someone-else' GITHUB_SHA="${release_sha}" "${validator}"
assert_fails env REPOSITORY_ROOT="${repository}" GITHUB_REF_NAME='v1.0.0-alpha.01' GITHUB_ACTOR='coval-release-automation[bot]' GITHUB_SHA="${release_sha}" "${validator}"

printf 'unexpected\n' > "${repository}/README.md"
git -C "${repository}" add README.md
git -C "${repository}" commit --quiet --message 'chore(release): 1.0.1'
extra_file_sha="$(git -C "${repository}" rev-parse HEAD)"
git -C "${repository}" push --quiet origin main
assert_fails env REPOSITORY_ROOT="${repository}" GITHUB_REF_NAME='v1.0.1' GITHUB_ACTOR='coval-release-automation[bot]' GITHUB_SHA="${extra_file_sha}" "${validator}"

echo 'release tag validation tests passed'
