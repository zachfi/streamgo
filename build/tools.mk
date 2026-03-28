#
# build/tools.mk - Containerized tool execution (no host install)
#
# Tools (lint, proto, jsonnet, etc.) run inside the tools Docker image via
# TOOLS_CMD. Nothing is installed on the user's system with go install.
#
# Prerequisite: build or pull the tools image once:
#   make tools-image-build   # or make tools-image-pull
#
# Usage in other fragments: run any tool as $(TOOLS_CMD) <tool> [args...]
# Example: $(TOOLS_CMD) golangci-lint run
#          $(TOOLS_CMD) buf generate
#
# RUN_TOOL is set to $(TOOLS_CMD) so targets can use $(RUN_TOOL) <tool> [args]
# and stay agnostic of the exact docker invocation.
#
# Override TOOLS_IMAGE / TOOLS_IMAGE_TAG in your Makefile or env if needed.
# Set registry= (in Makefile or on the command line) to use the registry image for all
# tool runs: TOOLS_CMD, tools-docker, tools-image-build, tools-image-push, tools-image-pull.
# Use registry= (empty) to use the default Docker Hub/local image.
# Set USE_LOCAL_TOOLS=1 only if you want to install and run tools on the host
# (then RUN_TOOL is empty and targets that use it will run the local binary).
#

GO               ?= go
VENDOR_CMD       ?= $(GO) mod tidy
GO_MOD_OUTDATED  ?= go-mod-outdated

TOOL_DIR     ?= tools
TOOL_CONFIG  ?= $(TOOL_DIR)/tools.go

TOOLS_IMAGE      ?= zachfi/streamgo-ci-tools
TOOLS_IMAGE_TAG  ?= latest
TOOLS_MOUNT_PATH ?= /tools

# When registry is set, run uses the registry image (so make drone, make lint use the same image as CI).
TOOLS_IMAGE_FULL = $(if $(registry),$(registry)/$(TOOLS_IMAGE),$(TOOLS_IMAGE))
TOOLS_CMD = docker run --rm -t -v $(abspath .):$(TOOLS_MOUNT_PATH) -w $(TOOLS_MOUNT_PATH) $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG)

# Default: run tools in container. Set USE_LOCAL_TOOLS=1 to run on host (requires `make tools` first).
RUN_TOOL := $(TOOLS_CMD)
ifeq ($(USE_LOCAL_TOOLS),1)
RUN_TOOL :=
endif

GOTOOLS ?= $(shell cd $(TOOL_DIR) 2>/dev/null && $(GO) list -e -f '{{ .Imports }}' -tags tools 2>/dev/null | tr -d '[]')

# --- Tools module: tidy, verify, outdated, update (run from root: make tools-tidy, etc.) ---
.PHONY: tools-tidy tools-verify tools-outdated-list tools-update-outdated
tools-tidy:
	@$(MAKE) -C $(TOOL_DIR) tidy
tools-verify:
	@$(MAKE) -C $(TOOL_DIR) verify
# List modules with newer versions (built-in go list -u -m).
tools-outdated-list:
	@$(MAKE) -C $(TOOL_DIR) outdated
# Update all tools deps to latest, then tidy. Run tools-tidy after if needed.
tools-update-outdated:
	@$(MAKE) -C $(TOOL_DIR) update

# --- Optional: install tools on host (only if USE_LOCAL_TOOLS=1) ---
.PHONY: tools tools-outdated tools-update tools-update-mod
tools:
	@echo "=== $(PROJECT_NAME) === [ tools            ]: Installing tools on host (use tools image instead for no host install)..."
	@cd $(TOOL_DIR) && $(GO) install $(GOTOOLS)
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

tools-outdated:
	@cd $(TOOL_DIR) && $(GO) list -u -m -json all | $(GO_MOD_OUTDATED) -direct -update

tools-update:
	@cd $(TOOL_DIR) && for x in $(GOTOOLS); do $(GO) get -u $$x; done
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

tools-update-mod:
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

# --- Tools container image ---
# All targets below use TOOLS_IMAGE_FULL (registry/TOOLS_IMAGE when registry is set).
# Pass registry= on the command line to override (e.g. make tools-docker registry=myreg:5000 or registry= for local).
.PHONY: tools-image-build tools-image-push tools-image-pull tools-docker
# Optional: alpine_mirror=https://... to use a different Alpine mirror (container DNS can differ from host).
tools-image-build:
	@echo "=== $(PROJECT_NAME) === [ tools-image-build ]: Building tools image..."
	@docker build $(if $(alpine_mirror),--build-arg ALPINE_MIRROR=$(alpine_mirror),) -t $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG) -f $(TOOL_DIR)/Dockerfile .
	@if [ -n "$(registry)" ]; then docker tag $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG) $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG); fi

tools-image-push:
	@echo "=== $(PROJECT_NAME) === [ tools-image-push  ]: Pushing tools image..."
	@docker push $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG)

tools-image-pull:
	@echo "=== $(PROJECT_NAME) === [ tools-image-pull  ]: Pulling tools image..."
	@docker pull $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG)
	@if [ -n "$(registry)" ]; then docker tag $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG) $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG); fi

# Shell in tools container. Uses registry image when registry= is set (e.g. make tools-docker or make tools-docker registry=myreg:5000).
tools-docker:
	@echo "=== $(PROJECT_NAME) === [ tools-docker      ]: Shell in tools container (project at $(TOOLS_MOUNT_PATH))..."
	@docker run -it -v $(abspath .):$(TOOLS_MOUNT_PATH) -w $(TOOLS_MOUNT_PATH) $(TOOLS_IMAGE_FULL):$(TOOLS_IMAGE_TAG) $(SHELL)