.PHONY: docs docs-host snippet-dependencies-check docs-snippets docs-snippets-check docs-check

DOCS_PORT ?= 3000
MINT_VERSION ?= 4.2.687

docs:
	docker run --rm -it \
		-p $(DOCS_PORT):$(DOCS_PORT) \
		-v "$(CURDIR):/work" \
		-w /work \
		node:22-bookworm \
		sh -lc "npx --yes mint@$(MINT_VERSION) dev --port $(DOCS_PORT) --host 0.0.0.0"
docs-host:
	npx --yes mint@$(MINT_VERSION) dev

snippet-dependencies-check:
	bash scripts/check-snippet-dependencies.sh

# Regenerate command-reference snippets from the local sitectl + plugin
# checkouts. The generator is a self-contained Go module that resolves the
# workspace repos under ../../cli through the replace directives in
# scripts/gen-docs-snippets/go.mod. GOWORK=off prevents an ambient workspace
# from replacing that graph. The generator runs from inside its module dir and
# is passed the docs root as the output base.
docs-snippets:
	cd scripts/gen-docs-snippets && GOWORK=off go run . "$(CURDIR)"

docs-snippets-check: snippet-dependencies-check docs-snippets
	git diff --exit-code -- snippets/commands
	@test -z "$$(git ls-files --others --exclude-standard -- snippets/commands)" || { \
		git status --short -- snippets/commands; \
		echo "generated command snippets are not committed" >&2; \
		exit 1; \
	}

docs-check:
	npx --yes mint@$(MINT_VERSION) validate
	npx --yes mint@$(MINT_VERSION) broken-links --check-anchors --check-redirects --check-snippets
