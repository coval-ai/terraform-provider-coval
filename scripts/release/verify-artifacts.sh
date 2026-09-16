#!/usr/bin/env bash

set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "${script_directory}/../.." && pwd)"
distribution_directory="${repository_root}/dist"

shopt -s nullglob
checksum_files=("${distribution_directory}"/terraform-provider-coval_*_SHA256SUMS)
linux_amd64_archives=("${distribution_directory}"/terraform-provider-coval_*_linux_amd64.zip)

if [[ "${#checksum_files[@]}" -ne 1 ]]; then
  echo "expected exactly one checksum file in ${distribution_directory}" >&2
  exit 1
fi
if [[ "${#linux_amd64_archives[@]}" -ne 1 ]]; then
  echo "expected exactly one linux_amd64 provider archive in ${distribution_directory}" >&2
  exit 1
fi

checksum_file="${checksum_files[0]}"
manifest_entries="$(awk '$2 ~ /^terraform-provider-coval_.*_manifest\.json$/ { print $2 }' "${checksum_file}")"
manifest_entry_count="$(printf '%s\n' "${manifest_entries}" | awk 'NF { count++ } END { print count + 0 }')"
if [[ "${manifest_entry_count}" -ne 1 ]]; then
  echo "expected exactly one Terraform Registry manifest in the checksum file" >&2
  exit 1
fi

source_manifest="${repository_root}/terraform-registry-manifest.json"
if [[ ! -f "${source_manifest}" ]]; then
  echo "Terraform Registry manifest not found: ${source_manifest}" >&2
  exit 1
fi

# GoReleaser's publish skip prevents release.extra_files from being staged, so
# copy the public manifest into the snapshot only long enough to verify the
# complete checksum set that a real release will publish.
snapshot_manifest="${distribution_directory}/${manifest_entries}"
if [[ -e "${snapshot_manifest}" ]]; then
  echo "snapshot manifest already exists: ${snapshot_manifest}" >&2
  exit 1
fi
cp -- "${source_manifest}" "${snapshot_manifest}"
trap 'rm -f -- "${snapshot_manifest}"' EXIT

if command -v sha256sum >/dev/null 2>&1; then
  (cd -- "${distribution_directory}" && sha256sum --check "$(basename -- "${checksum_file}")")
elif command -v shasum >/dev/null 2>&1; then
  (cd -- "${distribution_directory}" && shasum -a 256 -c "$(basename -- "${checksum_file}")")
else
  echo "sha256sum or shasum is required to verify release artifacts" >&2
  exit 1
fi

rm -f -- "${snapshot_manifest}"
trap - EXIT

echo 'release snapshot artifacts verified'
