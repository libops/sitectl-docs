module github.com/libops/sitectl-docs/gen-docs-snippets

go 1.25.8

require (
	github.com/libops/sitectl v0.0.0
	github.com/libops/sitectl-drupal v0.0.0
	github.com/libops/sitectl-isle v0.0.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
)

replace (
	github.com/libops/sitectl => ../../../sitectl
	github.com/libops/sitectl-drupal => ../../../sitectl-drupal
	github.com/libops/sitectl-isle => ../../../sitectl-isle
)
