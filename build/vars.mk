#
# build/vars.mk - Project variables for the build system
#
# Override these in your project's Makefile (before including build/*.mk)
# or set in the environment. This file provides defaults so the build
# partials work without much configuration.
#
# === Required / commonly overridden ===
# PROJECT_NAME   - Display name (default: directory name)
# IMG           - Primary Docker image (e.g. user/app:tag)
# LATESTIMG     - Docker image for :latest (e.g. user/app:latest)
#
# === Paths ===
# SRCDIR        - Source root (default: .)
# BUILD_DIR     - Where to put binaries (default: ./bin/)
# TOOL_DIR      - Directory with tools.go for go install (default: tools)
# DIST_DIR      - Release output (default: ./dist)
#
# === Version / tagging ===
# VERSION       - Version string for images (default: from tools/image-tag or git)
# PROJECT_VER   - Full version (default: git describe)
# PROJECT_VER_TAGGED - Last tagged version (default: git describe --abbrev=0)
#
# === Go ===
# GO            - Go command (default: go)
# PROJECT_MODULE - Go module path (default: go list -m)
#
# === Optional / feature-specific ===
# registry      - Docker registry prefix (set for docker-build/docker-push)
# RELEASE_SCRIPT - Script for release target (default: ./scripts/release.sh)
# CHANGELOG_CMD, CHANGELOG_FILE, etc. - See document.mk
#

GO             ?= go
SRCDIR         ?= .
BUILD_DIR      ?= ./bin/

# Native platform: last two fields of "go version" are always GOOS and GOARCH
# (handles non-standard version strings e.g. "go1.25.7 X:nodwarf5 linux/amd64")
NATIVEOS   ?= $(shell $(GO) version 2>/dev/null | awk -F'[ /]' '{print $$(NF-1)}')
NATIVEARCH ?= $(shell $(GO) version 2>/dev/null | awk -F'[ /]' '{print $$NF}')
GOOS       ?= $(NATIVEOS)
GOARCH     ?= $(NATIVEARCH)
DIST_DIR       ?= ./dist
TOOL_DIR       ?= tools
TOOL_CONFIG    ?= $(TOOL_DIR)/tools.go

# Project identity
PROJECT_NAME   ?= $(shell basename $(shell pwd))
PROJECT_MODULE ?= $(shell $(GO) list -m 2>/dev/null || echo "")
PROJECT_VER    ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed -e '/^v/s/^v\(.*\)$$/\1/g' || echo "dev")
PROJECT_VER_TAGGED ?= $(shell git describe --tags --always --abbrev=0 2>/dev/null | sed -e '/^v/s/^v\(.*\)$$/\1/g' || echo "dev")

# Docker: set IMG and LATESTIMG in your Makefile (e.g. IMG ?= user/myapp:$(VERSION))
# If tools/image-tag exists, VERSION can be derived for default IMG.
VERSION        ?= $(shell [ -x $(SRCDIR)/$(TOOL_DIR)/image-tag ] && $(SRCDIR)/$(TOOL_DIR)/image-tag | cut -d, -f1 || echo $(PROJECT_VER))
IMG            ?= $(PROJECT_NAME):$(VERSION)
LATESTIMG      ?= $(PROJECT_NAME):latest
DOCKER         ?= docker
