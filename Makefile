.PHONY: docs docs-host docs-snippets

DOCS_PORT ?= 3000

docs:
	docker run --rm -it \
		-p $(DOCS_PORT):$(DOCS_PORT) \
		-v "$(CURDIR):/work" \
		-w /work \
		node:22-bookworm \
		sh -lc "npx mint dev --port $(DOCS_PORT) --host 0.0.0.0"
docs-host:
	npx mint dev

# Regenerate command-reference snippets from the local sitectl + plugin
# checkouts. The generator is a self-contained Go module that resolves the
# sibling repos through the replace directives in
# scripts/gen-docs-snippets/go.mod, so no go.work file is needed. It runs from
# inside its module dir and is passed the docs root as the output base.
docs-snippets:
	cd scripts/gen-docs-snippets && go run . "$(CURDIR)"
