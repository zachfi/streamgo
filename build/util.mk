#
# Makefile fragment for utility items
#

NATIVEOS    ?= $(shell go version | awk -F'[ /]' '{print $$(NF-1)}')
NATIVEARCH  ?= $(shell go version | awk -F'[ /]' '{print $$NF}')


check-version:
ifdef GOOS
ifneq "$(GOOS)" "$(NATIVEOS)"
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
else
GOOS = ${NATIVEOS}
endif
ifdef GOARCH
ifneq "$(GOARCH)" "$(NATIVEARCH)"
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
else
GOARCH = ${NATIVEARCH}
endif

# No-op for projects that don't use build/proto.mk (proto.mk overrides when included)
proto:
	@true

# No-op for projects that don't add commit linting (e.g. build/commitlint.mk overrides when included)
lint-commit:
	@true

# Ensure host Go version satisfies go.mod (avoids "version X does not match go tool version Y" in tests)
check-go:
	@CURRENT=$$($(GO) version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1 | sed 's/^go//' | tr -d '\r'); \
	REQUIRED=$$(grep '^go ' go.mod 2>/dev/null | awk '{print $$2}' | tr -d '\r'); \
	if [ -z "$$REQUIRED" ]; then echo "=== $(PROJECT_NAME) === [ check-go        ]: could not read go version from go.mod"; exit 1; fi; \
	MAX=$$(printf '%s\n%s\n' "$$CURRENT" "$$REQUIRED" | sort -V | tail -1); \
	if [ "$$MAX" != "$$CURRENT" ]; then \
		echo "=== $(PROJECT_NAME) === [ check-go        ]: go.mod requires Go $$REQUIRED; you have go$${CURRENT:-unknown}."; \
		echo "    Upgrade Go (https://go.dev/dl/) or run: GOTOOLCHAIN=auto make test"; \
		exit 1; \
	fi

.PHONY: check-version check-go proto lint-commit
