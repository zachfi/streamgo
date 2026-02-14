# streamgo - minimal example of unified build structure
#
# Include vars first so VERSION etc. are set; then override project-specific
# variables. See build/README.md for variables and optional includes.

#############################
# Core (always include vars first)
#############################
include build/vars.mk

#############################
# Project-specific overrides
#############################
IMG       ?= zachfi/streamgo:$(VERSION)
LATESTIMG ?= zachfi/streamgo:latest

#############################
# Targets
#############################
all: build

# Humans: full local build
build: check-version clean lint test cover-report proto gofmt-fix compile

# CI: no coverage report, no proto
build-ci: check-version clean lint test compile-only

clean: cover-clean compile-clean release-clean

#############################
# Build partials (order can matter for overrides)
#############################
include build/util.mk
include build/tools.mk
include build/deps.mk
include build/compile.mk
include build/docker.mk
include build/document.mk
include build/lint.mk
include build/release.mk
include build/test.mk

# Optional: uncomment if this project had protos
# PROTO_GRPC_FILES = path/to/file.proto
# include build/proto.mk

# Optional: uncomment for operator/manager image build and push
# include build/kube_builder.mk

.PHONY: all build build-ci clean
