#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: $0 <version>" >&2
  exit 2
fi

version="$1"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="${REPOSITORY_ROOT:-$(cd -- "${script_directory}/../.." && pwd)}"

if ! git -C "${repository_root}" diff --cached --quiet --exit-code; then
  echo "release preparation must not contain staged changes" >&2
  exit 1
fi

worktree_status="$(git -C "${repository_root}" status --porcelain=v1 --untracked-files=all)"
if [[ "${worktree_status}" != ' M CHANGELOG.md' ]]; then
  echo "release preparation may only change CHANGELOG.md" >&2
  printf '%s\n' "${worktree_status}" >&2
  exit 1
fi

if ! grep -Fq -- "${version}" "${repository_root}/CHANGELOG.md"; then
  echo "CHANGELOG.md does not contain release version ${version}" >&2
  exit 1
fi

echo "Verified generated changelog for ${version}"
