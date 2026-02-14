# Build system (portable partials)

This directory contains Makefile fragments you can copy into other projects. The root `Makefile` stays small and only sets project-specific variables and includes the fragments it needs.

## Usage in a new project

1. Copy `build/` (and optionally `tools/` with `tools.go`) into the project.
2. In the project root `Makefile`:
   - Set any project variables (at least `IMG` and `LATESTIMG` for Docker).
   - Include `build/vars.mk` first.
   - Define high-level targets (`all`, `build`, `build-ci`, `clean`).
   - Include the build partials you need (see below).

Example minimal root Makefile:

```make
IMG       ?= user/myapp:$(VERSION)
LATESTIMG ?= user/myapp:latest

include build/vars.mk

all: build
build: check-version clean lint test cover-report proto gofmt-fix compile
build-ci: check-version clean lint test compile-only
clean: cover-clean compile-clean release-clean

include build/util.mk
include build/tools.mk
include build/deps.mk
include build/compile.mk
include build/docker.mk
include build/document.mk
include build/lint.mk
include build/release.mk
include build/test.mk
# include build/proto.mk      # if you have .proto files
# include build/kube_builder.mk # if you build an operator/manager image
```

## Variables (build/vars.mk)

Set these in your **Makefile** (before `include build/vars.mk`) or in the environment.

| Variable | Default | Description |
|----------|---------|-------------|
| `PROJECT_NAME` | `basename $(pwd)` | Display name in log lines |
| `IMG` | `$(PROJECT_NAME):$(VERSION)` | Primary Docker image |
| `LATESTIMG` | `$(PROJECT_NAME):latest` | Docker `:latest` tag |
| `SRCDIR` | `.` | Source root |
| `BUILD_DIR` | `./bin/` | Binary output directory |
| `TOOL_DIR` | `tools` | Directory containing `tools.go` |
| `DIST_DIR` | `./dist` | Release output |
| `VERSION` | from `tools/image-tag` or git | Version for images |
| `PROJECT_VER` | `git describe ...` | Full version string |
| `PROJECT_MODULE` | `go list -m` | Go module path |
| `registry` | (none) | Set for `docker-build`/`docker-push` (e.g. `make docker-build registry=myreg.io`) |

## Build partials

| File | Purpose |
|------|---------|
| **vars.mk** | Always include first. Defines default variables. |
| **util.mk** | `check-version`, no-op `proto` and `lint-commit`. |
| **tools.mk** | Install tools from `$(TOOL_DIR)/tools.go`. |
| **deps.mk** | `deps`, `deps-only` (mod tidy, vendor). |
| **compile.mk** | Build binaries from `cmd/*` (`compile`, `compile-only`, `compile-clean`, etc.). |
| **docker.mk** | `docker-login`, `docker`, `docker-snapshot` (generic image). |
| **document.mk** | `docs`, `changelog`, `release-notes`. |
| **lint.mk** | `lint`, spell-check, gofmt, golangci, goimports, outdated. |
| **release.mk** | `release`, `release-clean`, `release-publish`, `snapshot`. |
| **test.mk** | `test`, `test-unit`, `test-integration`, coverage targets. |
| **proto.mk** | Optional. Protobuf/grpc codegen. Set `PROTO_GRPC_FILES` and optionally `PROTO_IMPORT_DIRS` before including. |
| **kube_builder.mk** | Optional. `docker-build` and `docker-push` for operator/manager image (uses `IMG`, `LATESTIMG`, `registry`). |

## Optional features

- **Protos**: Add `PROTO_GRPC_FILES = path/to/file.proto ...` and `include build/proto.mk`. If you don’t include `proto.mk`, the `proto` target is a no-op.
- **Operator/manager image**: Include `build/kube_builder.mk` for `docker-build` and `docker-push` targets. Omit for plain apps that only use `docker.mk`.
- **Commit lint**: Add a fragment that defines `lint-commit` (e.g. `build/commitlint.mk`) and include it; otherwise `lint-commit` is a no-op.

## Reference project

This repo (streamgo) is kept as a minimal example: small root Makefile, no proto, no kube_builder. Copy `build/` and adjust variables and includes for projects that need protos or operator builds.
