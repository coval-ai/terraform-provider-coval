#!/usr/bin/env bash

set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
validator="${script_directory}/validate-pr-title.sh"

assert_fails() {
  if "$@" >/dev/null 2>&1; then
    echo "expected command to fail: $*" >&2
    exit 1
  fi
}

"${validator}" 'feat: add a resource'
"${validator}" 'fix(test-set): preserve ordering'
"${validator}" 'feat!: replace a schema'
"${validator}" 'build(deps-dev): update release tooling'

assert_fails "${validator}" 'Add a resource'
assert_fails "${validator}" 'feat: '
assert_fails "${validator}" 'Feat: add a resource'
assert_fails "${validator}" 'fix(Test Set): preserve ordering'

echo 'pull request title tests passed'
