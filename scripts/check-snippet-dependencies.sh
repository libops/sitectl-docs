#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
manifest="$repo_root/scripts/snippet-dependencies.json"
legacy_manifest="$repo_root/scripts/legacy-snippet-dependencies.json"
generator="$repo_root/scripts/gen-docs-snippets"
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
  sitectl_libops
  sitectl_ojs
  sitectl_omeka_classic
  sitectl_omeka_s
  sitectl_wp
)
expected_keys_json="$(printf '%s\n' "${expected_keys[@]}" | jq -R . | jq -cs 'sort')"
legacy_expected_keys_json="$(
  printf '%s\n' sitectl sitectl_drupal sitectl_isle | jq -R . | jq -cs 'sort'
)"

validate_manifest() {
  local path="$1" expected="$2" description="$3"
  jq -e --argjson expected_keys "$expected" '
  type == "object" and
  (keys == $expected_keys) and
  all(to_entries[];
    (.value | type == "object") and
    (.value | keys == ["directory", "module", "ref", "repository", "version"]) and
    (.value.directory | test("^sitectl(-[a-z0-9]+)*$")) and
    (.key == (.value.directory | gsub("-"; "_"))) and
    (.value.module == ("github.com/libops/" + .value.directory)) and
    (.value.repository == ("libops/" + .value.directory)) and
    (.value.version | test("^v[0-9]+\\.[0-9]+\\.[0-9]+$")) and
    (.value.ref | test("^[0-9a-f]{40}$"))
  )
' "$path" >/dev/null || {
    echo "$description does not match the exact supported repository schema" >&2
    exit 1
  }
}

validate_manifest "$manifest" "$expected_keys_json" "active snippet dependency manifest"
validate_manifest "$legacy_manifest" "$legacy_expected_keys_json" "legacy ISLE dependency manifest"

if [[ "$mode" == "--manifest-only" ]]; then
  echo "Active and legacy snippet dependency manifest schemas are valid"
  exit 0
fi

go_mod_json="$(cd "$generator" && GOWORK=off go mod edit -json)"
module_pattern='^github\.com/libops/sitectl(-[a-z0-9]+)*$'

manifest_requires="$(jq -c '
  [.[] | {Path: .module, Version: .version, Indirect: false}] | sort_by(.Path)
' "$manifest")"
go_mod_requires="$(jq -c --arg pattern "$module_pattern" '
  [.Require[]?
    | select(.Path | test($pattern))
    | {Path, Version, Indirect: (.Indirect // false)}
  ] | sort_by(.Path)
' <<<"$go_mod_json")"
if [[ "$manifest_requires" != "$go_mod_requires" ]]; then
  echo "snippet manifest and generator sitectl require directives must match exactly" >&2
  diff -u \
    <(jq . <<<"$manifest_requires") \
    <(jq . <<<"$go_mod_requires") >&2 || true
  exit 1
fi

manifest_replaces="$(jq -c '
  [.[] | {
    Path: .module,
    OldVersion: "",
    NewPath: ("../../../../cli/" + .directory),
    NewVersion: ""
  }] | sort_by(.Path)
' "$manifest")"
go_mod_replaces="$(jq -c --arg pattern "$module_pattern" '
  [.Replace[]?
    | select(.Old.Path | test($pattern))
    | {
      Path: .Old.Path,
      OldVersion: (.Old.Version // ""),
      NewPath: (.New.Path // ""),
      NewVersion: (.New.Version // "")
    }
  ] | sort_by(.Path)
' <<<"$go_mod_json")"
if [[ "$manifest_replaces" != "$go_mod_replaces" ]]; then
  echo "snippet manifest and generator sitectl replace directives must match exactly" >&2
  diff -u \
    <(jq . <<<"$manifest_replaces") \
    <(jq . <<<"$go_mod_replaces") >&2 || true
  exit 1
fi

tag_errors=0
while IFS=$'\t' read -r repository version expected_ref; do
  if ! remote_refs="$(git ls-remote --exit-code \
    "https://github.com/${repository}.git" \
    "refs/tags/${version}" \
    "refs/tags/${version}^{}")"; then
    echo "snippet dependency $repository tag $version is not published or could not be resolved" >&2
    tag_errors=1
    continue
  fi
  resolved_ref="$(awk '
    $2 ~ /\^\{\}$/ { peeled = $1 }
    $2 !~ /\^\{\}$/ { direct = $1 }
    END { print peeled != "" ? peeled : direct }
  ' <<<"$remote_refs")"
  if [[ "$resolved_ref" != "$expected_ref" ]]; then
    echo "snippet dependency $repository tag $version resolves to $resolved_ref, expected $expected_ref" >&2
    tag_errors=1
  fi
done < <(
  jq -sr '
    .[]
    | to_entries[]
    | [.value.repository, .value.version, .value.ref]
    | @tsv
  ' "$manifest" "$legacy_manifest"
)

if ((tag_errors != 0)); then
  exit 1
fi

echo "Active snippet manifest, legacy ISLE manifest, module graph, and release tags are valid"
