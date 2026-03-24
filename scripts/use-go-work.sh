#!/usr/bin/env bash

set -euo pipefail

for repo in ../sitectl ../sitectl-isle ../sitectl-drupal; do
	if [[ ! -f "${repo}/go.mod" ]]; then
		echo "Skipping go.work; ${repo}/go.mod not found"
		exit 0
	fi
done

cat > go.work <<EOF
go 1.25.8

use (
	./scripts/gen-docs-snippets
	../sitectl
	../sitectl-isle
	../sitectl-drupal
)
EOF

echo "Wrote go.work"
