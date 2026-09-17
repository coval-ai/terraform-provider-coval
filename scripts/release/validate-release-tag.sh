#!/usr/bin/env bash

set -euo pipefail

: "${GITHUB_REF_NAME:?GITHUB_REF_NAME is required}"
: "${GITHUB_ACTOR:?GITHUB_ACTOR is required}"
: "${GITHUB_SHA:?GITHUB_SHA is required}"

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="${REPOSITORY_ROOT:-$(cd -- "${script_directory}/../.." && pwd)}"

semver_pattern='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
if [[ ! "${GITHUB_REF_NAME}" =~ ${semver_pattern} ]]; then
  echo "Release tags must be complete semantic versions prefixed with v." >&2
  exit 1
fi

version_without_prefix="${GITHUB_REF_NAME#v}"
if [[ "${version_without_prefix}" == *-* ]]; then
  prerelease="${version_without_prefix#*-}"
  prerelease="${prerelease%%+*}"
  IFS='.' read -r -a prerelease_identifiers <<< "${prerelease}"
  for identifier in "${prerelease_identifiers[@]}"; do
    if [[ "${identifier}" =~ ^[0-9]+$ && "${identifier}" != "0" && "${identifier}" == 0* ]]; then
      echo "Numeric prerelease identifiers must not contain leading zeroes." >&2
      exit 1
    fi
  done
fi

if [[ "${GITHUB_ACTOR}" != 'coval-release-automation[bot]' ]]; then
  echo "Release tags must be created by Coval Release Automation." >&2
  exit 1
fi

if ! git -C "${repository_root}" merge-base --is-ancestor "${GITHUB_SHA}" origin/main; then
  echo "Release tags must point to a commit in main history." >&2
  exit 1
fi

expected_subject="chore(release): ${version_without_prefix}"
actual_subject="$(git -C "${repository_root}" show --no-patch --format='%s' "${GITHUB_SHA}")"
if [[ "${actual_subject}" != "${expected_subject}" ]]; then
  echo "Release tag commit subject must be '${expected_subject}'." >&2
  exit 1
fi

changed_files="$(git -C "${repository_root}" diff-tree --no-commit-id --name-only -r "${GITHUB_SHA}")"
if [[ "${changed_files}" != 'CHANGELOG.md' ]]; then
  echo "Release tag commits may only change CHANGELOG.md." >&2
  exit 1
fi

echo "Verified release tag ${GITHUB_REF_NAME}"
