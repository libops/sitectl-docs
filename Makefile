.PHONY: docs

DOCS_PORT ?= 3000

docs:
	docker run --rm -it \
		-p $(DOCS_PORT):$(DOCS_PORT) \
		-v "$(CURDIR):/work" \
		-w /work \
		node:22-bookworm \
		sh -lc "npx mint dev --port $(DOCS_PORT) --host 0.0.0.0"

