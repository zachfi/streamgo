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

.PHONY: check-version proto lint-commit
