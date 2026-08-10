#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
manifest="$repo_root/scripts/snippet-dependencies.json"
generator="$repo_root/scripts/gen-docs-snippets"
programs="$repo_root/scripts/snippet-dependency-programs"
mode="${1:-}"

case "$mode" in
  "" | --manifest-only) ;;
  *)
    echo "usage: $0 [--manifest-only]" >&2
    exit 2
    ;;
esac

expected_keys=(
  sitectl
  sitectl_archivesspace
  sitectl_drupal
  sitectl_isle
  sitectl_libops
  sitectl_ojs
  sitectl_omeka_classic
  sitectl_omeka_s
  sitectl_wp
)
expected_keys_json="$(printf '%s\n' "${expected_keys[@]}" | jq -Rn -f "$programs/expected-keys.jq")"
jq -e --argjson expected_keys "$expected_keys_json" \
  -f "$programs/validate-manifest.jq" "$manifest" >/dev/null || {
  echo "snippet dependency manifest does not match the exact supported repository schema" >&2
  exit 1
}

if [[ "$mode" == "--manifest-only" ]]; then
  echo "Snippet dependency manifest schema is valid"
  exit 0
fi

go_mod_json="$(cd "$generator" && GOWORK=off go mod edit -json)"
module_pattern='^github\.com/libops/sitectl(-[a-z0-9]+)*$'
module_namespace_pattern='^github\.com/libops/sitectl($|[-/])'

manifest_requires="$(jq -c -f "$programs/manifest-requires.jq" "$manifest")"
go_mod_requires="$(jq -c \
  --arg namespace_pattern "$module_namespace_pattern" \
  --arg pattern "$module_pattern" \
  -f "$programs/go-mod-requires.jq" <<<"$go_mod_json")"
if [[ "$manifest_requires" != "$go_mod_requires" ]]; then
  echo "snippet manifest and generator sitectl require directives must match exactly" >&2
  diff -u \
    <(jq -f "$programs/pretty-print.jq" <<<"$manifest_requires") \
    <(jq -f "$programs/pretty-print.jq" <<<"$go_mod_requires") >&2 || true
  exit 1
fi

manifest_replaces="$(jq -c -f "$programs/manifest-replaces.jq" "$manifest")"
go_mod_replaces="$(jq -c \
  --arg namespace_pattern "$module_namespace_pattern" \
  --arg pattern "$module_pattern" \
  -f "$programs/go-mod-replaces.jq" <<<"$go_mod_json")"
if [[ "$manifest_replaces" != "$go_mod_replaces" ]]; then
  echo "snippet manifest and generator sitectl replace directives must match exactly" >&2
  diff -u \
    <(jq -f "$programs/pretty-print.jq" <<<"$manifest_replaces") \
    <(jq -f "$programs/pretty-print.jq" <<<"$go_mod_replaces") >&2 || true
  exit 1
fi

expected_dependency_count="${#expected_keys[@]}"
release_records_text="$(
  jq -er --argjson expected_count "$expected_dependency_count" \
    -f "$programs/release-records.jq" "$manifest"
)"
mapfile -t release_records <<<"$release_records_text"
if ((${#release_records[@]} != expected_dependency_count)); then
  echo "expected exactly $expected_dependency_count snippet dependency release records" >&2
  exit 1
fi

tag_errors=0
for record in "${release_records[@]}"; do
  IFS=$'\t' read -r repository version expected_ref extra <<<"$record"
  if [[ -z "$repository" || -z "$version" || -z "$expected_ref" || -n "$extra" ]]; then
    echo "snippet dependency release records must contain exactly three nonempty fields" >&2
    exit 1
  fi

  if ! remote_refs="$(git ls-remote --exit-code \
    "https://github.com/${repository}.git" \
    "refs/tags/${version}" \
    "refs/tags/${version}^{}")"; then
    echo "snippet dependency $repository tag $version is not published or could not be resolved" >&2
    tag_errors=1
    continue
  fi
  resolved_ref="$(awk -f "$programs/resolve-tag-ref.awk" <<<"$remote_refs")"
  if [[ "$resolved_ref" != "$expected_ref" ]]; then
    echo "snippet dependency $repository tag $version resolves to $resolved_ref, expected $expected_ref" >&2
    tag_errors=1
  fi
done

if ((tag_errors != 0)); then
  exit 1
fi

echo "Snippet dependency manifest, module graph, and release tags are valid"
