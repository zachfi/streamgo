#
# build/ci.mk - CI pipeline generation and local CI run
#
# make ci         - Generate pipeline YAML and run the same steps CI runs (compile, test).
# make ci-pipeline - Only generate .woodpecker.yml from build/woodpecker.jsonnet (runs jsonnet in tools container).
#
# Override in your Makefile or env:
#   CI_CONFIG         - Generated pipeline YAML (default: .woodpecker.yml)
#   CI_JSONNET_SOURCE - Jsonnet source (default: build/woodpecker.jsonnet)
#
# Expects build/tools.mk (RUN_TOOL, TOOLS_IMAGE_FULL, etc.).
# Prerequisite: make tools-image-build (or tools-image-pull) so the tools image has jsonnet and jq.
#

CI_CONFIG         ?= .woodpecker.yml
CI_JSONNET_SOURCE ?= build/woodpecker.jsonnet

.PHONY: ci ci-pipeline

# Generate pipeline YAML from jsonnet (runs in tools container: jsonnet | jq -r . > config)
ci-pipeline:
	@$(RUN_TOOL) sh -c 'jsonnet $(CI_JSONNET_SOURCE) | jq -r . > $(CI_CONFIG)'
	@echo "=== $(PROJECT_NAME) === [ ci-pipeline     ]: wrote $(CI_CONFIG)"

# Generate pipeline, then run the same build/test steps CI runs (compile + test).
# check-go ensures host Go satisfies go.mod so tests don't fail with version mismatch.
ci: ci-pipeline check-go compile-only test-only
	@echo "=== $(PROJECT_NAME) === [ ci              ]: pipeline generated and local CI steps passed"
