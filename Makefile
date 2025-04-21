MKDOCS ?= mkdocs
ADDR ?= localhost:8000

.PHONY: docs-build docs-serve docs-deploy

docs-build:
	$(MKDOCS) build --clean --strict

docs-serve:
	$(MKDOCS) serve --dev-addr $(ADDR)

docs-deploy:
	$(MKDOCS) gh-deploy --force
