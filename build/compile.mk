#
# Makefile fragment for compiling Go commands (cmd/*)
#
# Expects build/vars.mk (or equivalent) to set: GO, SRCDIR, BUILD_DIR,
# PROJECT_NAME, PROJECT_VER, PROJECT_MODULE, GOOS, GOARCH.
#

# Reject invalid GOOS from environment (e.g. GOOS=X:nodwarf5 from mistaken export).
# override is required so we win over GOOS set in the environment.
VALID_GOOS := linux darwin windows freebsd netbsd openbsd plan9 android
ifneq ($(filter $(GOOS),$(VALID_GOOS)),)
# GOOS is valid, use it
else
override GOOS := $(NATIVEOS)
endif

# $b replaced by the binary name in the compile loop; -s/-w strip debug symbols
LDFLAGS    ?= "-s -w -X main.Version=$(PROJECT_VER) -X main.appName=$$b"
COMPILE_OS ?= freebsd linux

# Commands from cmd/*
COMMANDS   ?= $(wildcard $(SRCDIR)/cmd/*)
BINS       := $(foreach cmd,$(COMMANDS),$(notdir $(cmd)))

compile-clean:
	@echo "=== $(PROJECT_NAME) === [ compile-clean    ]: removing binaries..."
	@rm -rfv $(BUILD_DIR)/*

compile: deps compile-only

compile-all: deps-only
	@echo "=== $(PROJECT_NAME) === [ compile          ]: building commands:"
	@mkdir -p $(BUILD_DIR)/$(GOOS)
	@for b in $(BINS); do \
		for os in $(COMPILE_OS); do \
			echo "=== $(PROJECT_NAME) === [ compile          ]:     $(BUILD_DIR)$$os/$$b"; \
			BUILD_FILES=`find $(SRCDIR)/cmd/$$b -type f -name "*.go"` ; \
			GOOS=$$os $(GO) build -ldflags=$(LDFLAGS) -o $(BUILD_DIR)/$$os/$$b $$BUILD_FILES ; \
		done \
	done

compile-only: deps-only
	@echo "=== $(PROJECT_NAME) === [ compile          ]: building commands:"
	@mkdir -p $(BUILD_DIR)/$(GOOS)
	@for b in $(BINS); do \
		echo "=== $(PROJECT_NAME) === [ compile          ]:     $(BUILD_DIR)$(GOOS)/$$b"; \
		BUILD_FILES=`find $(SRCDIR)/cmd/$$b -type f -name "*.go"` ; \
		CGO_ENABLED=0 GOOS=$(GOOS) $(GO) build -ldflags=$(LDFLAGS) -o $(BUILD_DIR)/$(GOOS)/$$b $$BUILD_FILES ; \
	done

compile-darwin: GOOS=darwin
compile-darwin: deps-only compile-only

compile-linux: GOOS=linux
compile-linux: deps-only compile-only

compile-freebsd: GOOS=freebsd
compile-freebsd: deps-only compile-only

.PHONY: compile-clean compile compile-all compile-only compile-darwin compile-linux compile-freebsd
