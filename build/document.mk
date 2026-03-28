#
# Makefile fragment for displaying auto-generated documentation
#
# Commands run in the tools container via RUN_TOOL (see build/tools.mk).
#

GODOC       ?= godoc
GODOC_HTTP  ?= "localhost:6060"
CHANGELOG_CMD      ?= git-chglog
CHANGELOG_FILE     ?= CHANGELOG.md
RELEASE_NOTES_FILE ?= relnotes.md

docs:
	@echo "=== $(PROJECT_NAME) === [ docs             ]: Starting godoc server..."
	@echo "=== $(PROJECT_NAME) === [ docs             ]: Module Docs: http://$(GODOC_HTTP)/pkg/$(PROJECT_MODULE)"
	@$(RUN_TOOL) $(GODOC) -http=$(GODOC_HTTP)

changelog:
	@echo "=== $(PROJECT_NAME) === [ changelog        ]: Generating changelog..."
	@$(RUN_TOOL) $(CHANGELOG_CMD) --silent -o $(CHANGELOG_FILE)

release-notes:
	@echo "=== $(PROJECT_NAME) === [ release-notes    ]: Generating release notes..."
	@mkdir -p $(SRCDIR)/tmp
	@$(RUN_TOOL) $(CHANGELOG_CMD) --silent -o $(SRCDIR)/tmp/$(RELEASE_NOTES_FILE) v$(PROJECT_VER_TAGGED)

.PHONY: docs changelog release-notes
