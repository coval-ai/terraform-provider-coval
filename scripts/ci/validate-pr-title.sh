#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: $0 <pull-request-title>" >&2
  exit 2
fi

pull_request_title="$1"
title_pattern='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9][a-z0-9._/-]*\))?(!)?: [^[:space:]].*$'

if [[ ! "${pull_request_title}" =~ ${title_pattern} ]]; then
  echo "Pull request titles must use Conventional Commit format." >&2
  echo "Examples: feat: add a resource; fix(test-set): preserve ordering; feat!: replace a schema" >&2
  exit 1
fi
