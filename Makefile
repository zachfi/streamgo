# streamgo - minimal example of unified build structure
#
# Include vars first, then tools.mk so targets that need tools run them in
# Docker (no go install on the host). Run "make tools-image-build" once.
# See build/README.md for variables and optional includes.

#############################
# Core (vars first, then tools so TOOLS_CMD is available)
#############################
include build/vars.mk
include build/tools.mk

#############################
# Project-specific overrides
#############################
# Use registry tools image so make ci, make lint, etc. use the same image as CI.
# Override with registry= to use a different registry, or registry= (empty) to use Docker Hub/local.
registry ?= reg.dist.svc.cluster.znet:5000

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
include build/deps.mk
include build/compile.mk
include build/docker.mk
include build/document.mk
include build/lint.mk
include build/release.mk
include build/test.mk
include build/ci.mk

# Optional: uncomment if this project had protos
# PROTO_GRPC_FILES = path/to/file.proto
# include build/proto.mk

# Optional: uncomment for operator/manager image build and push
# include build/kube_builder.mk

.PHONY: all build build-ci ci ci-pipeline clean
