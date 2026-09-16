#!/usr/bin/env bash

set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
validator="${script_directory}/verify-generated-changelog.sh"
temporary_directory="$(mktemp -d)"
trap 'rm -rf -- "${temporary_directory}"' EXIT

assert_fails() {
  if "$@" >/dev/null 2>&1; then
    echo "expected command to fail: $*" >&2
    exit 1
  fi
}

create_repository() {
  local name="$1"
  local repository="${temporary_directory}/${name}"

  mkdir -p "${repository}"
  git -C "${repository}" init --quiet
  git -C "${repository}" config user.name 'Release Test'
  git -C "${repository}" config user.email 'release-test@example.com'
  git -C "${repository}" config commit.gpgsign false
  printf '# Changelog\n' > "${repository}/CHANGELOG.md"
  git -C "${repository}" add CHANGELOG.md
  git -C "${repository}" commit --quiet --message 'initial changelog'

  printf '%s\n' "${repository}"
}

valid_repository="$(create_repository valid)"
printf '# Changelog\n\n## 1.0.0\n' > "${valid_repository}/CHANGELOG.md"
REPOSITORY_ROOT="${valid_repository}" "${validator}" '1.0.0' >/dev/null

extra_file_repository="$(create_repository extra-file)"
printf '# Changelog\n\n## 1.0.0\n' > "${extra_file_repository}/CHANGELOG.md"
printf 'unexpected\n' > "${extra_file_repository}/README.md"
assert_fails env REPOSITORY_ROOT="${extra_file_repository}" "${validator}" '1.0.0'

staged_repository="$(create_repository staged)"
printf '# Changelog\n\n## 1.0.0\n' > "${staged_repository}/CHANGELOG.md"
git -C "${staged_repository}" add CHANGELOG.md
assert_fails env REPOSITORY_ROOT="${staged_repository}" "${validator}" '1.0.0'

missing_version_repository="$(create_repository missing-version)"
printf '# Changelog\n\nNo release heading here.\n' > "${missing_version_repository}/CHANGELOG.md"
assert_fails env REPOSITORY_ROOT="${missing_version_repository}" "${validator}" '1.0.0'

echo 'generated changelog tests passed'
