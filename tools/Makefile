# tools/Makefile - Go module tasks for the tools package
#
# Run from repo root: make -C tools tidy
# Or from tools/: make tidy
#

GO ?= go

.PHONY: tidy verify download outdated update

# Prune go.sum and update go.mod to match imports; run after editing tools.go or go.mod.
tidy:
	$(GO) mod tidy

# Verify dependencies have expected checksums.
verify: tidy
	$(GO) mod verify

# Download modules to cache (e.g. before tools-image-build).
download:
	$(GO) mod download

# List modules that have newer versions available (built-in; no extra tools).
outdated:
	$(GO) list -u -m all

# Update all dependencies to their latest versions, then tidy.
update:
	$(GO) get -u ./...
	$(GO) mod tidy
