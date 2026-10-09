#!/usr/bin/env bash
# Construct commits only in disposable test repositories, reusing source metadata.
set -euo pipefail
repository="$1"
message="$2"
source_root="$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
tree="$(git -C "${repository}" write-tree)"
parent="$(git -C "${repository}" rev-parse --verify HEAD 2>/dev/null || true)"
commit="$({
  printf 'tree %s\n' "${tree}"
  if [[ -n "${parent}" ]]; then printf 'parent %s\n' "${parent}"; fi
  git -C "${source_root}" cat-file commit HEAD | sed -n '/^author /p; /^committer /p'
  printf '\n%s\n' "${message}"
} | git -C "${repository}" hash-object -t commit -w --stdin)"
git -C "${repository}" update-ref HEAD "${commit}"
