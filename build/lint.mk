#
# Makefile fragment for Linting
#
# Linters run in the tools container via RUN_TOOL (see build/tools.mk).
# No tools are installed on the host.
#

GO           ?= go
MISSPELL     ?= misspell
GOFMT        ?= gofmt
GOIMPORTS    ?= goimports
GOLINTER     ?= golangci-lint
GO_MOD_OUTDATED ?= go-mod-outdated

EXCLUDEDIR   ?= .git
SRCDIR       ?= .
GO_PKGS      ?= $(shell $(GO) list ./... | grep -v -e "/vendor/" -e "/example")
FILES        ?= $(shell find $(SRCDIR) -type f | grep -v -e '.git/' -e '/vendor/')
GO_FILES     ?= $(shell find $(SRCDIR) -type f -name "*.go" | grep -v -e ".git/" -e '/vendor/' -e '/example/')
PROJECT_MODULE ?= $(shell $(GO) list -m 2>/dev/null)

lint: deps spell-check gofmt lint-commit golangci goimports outdated
lint-fix: deps spell-check-fix gofmt-fix goimports

spell-check: deps
	@echo "=== $(PROJECT_NAME) === [ spell-check      ]: Checking for spelling mistakes with $(MISSPELL)..."
	@$(RUN_TOOL) $(MISSPELL) -source text $(FILES)

spell-check-fix: deps
	@echo "=== $(PROJECT_NAME) === [ spell-check-fix  ]: Fixing spelling mistakes with $(MISSPELL)..."
	@$(RUN_TOOL) $(MISSPELL) -source text -w $(FILES)

gofmt: deps
	@echo "=== $(PROJECT_NAME) === [ gofmt            ]: Checking file format with $(GOFMT)..."
	@$(RUN_TOOL) gofmt -e -l -s -d $(GO_FILES)

gofmt-fix: deps
	@echo "=== $(PROJECT_NAME) === [ gofmt-fix        ]: Fixing file format with $(GOFMT)..."
	@$(RUN_TOOL) gofmt -e -l -s -w $(GO_FILES)

goimports: deps
	@echo "=== $(PROJECT_NAME) === [ goimports        ]: Checking imports with $(GOIMPORTS)..."
	@$(RUN_TOOL) $(GOIMPORTS) -l -w -local $(PROJECT_MODULE) $(GO_FILES)

golangci: deps
	@echo "=== $(PROJECT_NAME) === [ golangci-lint    ]: Linting using $(GOLINTER)"
	@$(RUN_TOOL) $(GOLINTER) run

outdated: deps
	@echo "=== $(PROJECT_NAME) === [ outdated         ]: Finding outdated deps with $(GO_MOD_OUTDATED)..."
	@$(RUN_TOOL) sh -c '$(GO) list -u -m -json all | $(GO_MOD_OUTDATED) -direct -update'

.PHONY: lint spell-check spell-check-fix gofmt gofmt-fix lint-fix lint-commit outdated goimports golangci
