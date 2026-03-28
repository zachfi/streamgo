# Build system (portable partials)

This directory contains Makefile fragments you can copy into other projects. The root `Makefile` stays small and only sets project-specific variables and includes the fragments it needs.

## Usage in a new project

1. Copy `build/` and `tools/` (with `tools/Dockerfile` and `tools/install.sh`) into the project.
2. In the project root `Makefile`: include `build/vars.mk` first, then `build/tools.mk` so `TOOLS_CMD` is available. Set project variables, define targets, and include the build partials you need.
3. Build the tools image once: `make tools-image-build` (or pull it). No `go install` on the host; lint, proto, etc. run in the container.

Example minimal root Makefile:

```make
include build/vars.mk
include build/tools.mk

IMG       ?= user/myapp:$(VERSION)
LATESTIMG ?= user/myapp:latest

all: build
build: check-version clean lint test cover-report proto gofmt-fix compile
build-ci: check-version clean lint test compile-only
clean: cover-clean compile-clean release-clean

include build/util.mk
include build/deps.mk
include build/compile.mk
include build/docker.mk
include build/document.mk
include build/lint.mk
include build/release.mk
include build/test.mk
include build/ci.mk
# include build/proto.mk      # if you use buf/protobuf
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
| **tools.mk** | Defines `TOOLS_CMD` and `RUN_TOOL`; tools run in Docker by default (no host install). Include right after vars.mk. |
| **deps.mk** | `deps`, `deps-only` (mod tidy, vendor). |
| **compile.mk** | Build binaries from `cmd/*` (`compile`, `compile-only`, `compile-clean`, etc.). |
| **docker.mk** | `docker-login`, `docker`, `docker-snapshot` (generic image). |
| **document.mk** | `docs`, `changelog`, `release-notes`. |
| **lint.mk** | `lint`, spell-check, gofmt, golangci, goimports, outdated. |
| **release.mk** | `release`, `release-clean`, `release-publish`, `snapshot`. |
| **test.mk** | `test`, `test-unit`, `test-integration`, coverage targets. |
| **proto.mk** | Optional. Runs `buf build`, `buf lint`, `buf generate` in the tools container. |
| **ci.mk** | CI pipeline: generate `.woodpecker.yml` from `build/woodpecker.jsonnet`; `make ci` runs pipeline generation plus compile and test locally. |
| **kube_builder.mk** | Optional. `docker-build` and `docker-push` for operator/manager image (uses `IMG`, `LATESTIMG`, `registry`). |

## Tools in Docker (tools.mk)

Tools (lint, proto, jsonnet, drone, etc.) run **inside the tools Docker image** by default. Nothing is installed on the host with `go install`.

1. **Build or pull the tools image once**: `make tools-image-build` (uses `tools/Dockerfile`) or `make tools-image-pull`.
2. **Run as usual**: `make lint`, `make proto`, `make ci`, etc. They invoke `$(TOOLS_CMD) <tool> ...` so the tool runs in the container.

Variables (override in Makefile or env):

| Variable | Default | Description |
|----------|---------|--------------|
| `TOOLS_IMAGE` | `zachfi/streamgo-ci-tools` | Docker image that provides the tools |
| `TOOLS_IMAGE_TAG` | `latest` | Tag for the tools image |
| `TOOLS_MOUNT_PATH` | `/tools` | Path in the container where the project is mounted |
| `USE_LOCAL_TOOLS` | `0` | Set to `1` to run tools on the host (requires `make tools` first) |

Use `$(TOOLS_CMD) <tool> [args...]` in any target that needs a tool, e.g. `$(TOOLS_CMD) buf generate`. Fragments like **lint.mk** and **proto.mk** use `RUN_TOOL` (which equals `TOOLS_CMD` by default) so they never touch the host.

**Registry for tool runs**: When `registry` is set (e.g. in your Makefile with `registry ?= reg.dist.svc.cluster.znet:5000`), all tools-image and tool-run targets use the registry image: `TOOLS_CMD` (so `make ci`, `make lint`, etc.), `tools-docker`, `tools-image-build`, `tools-image-push`, and `tools-image-pull`. Use the same image as CI and avoid an old local image. Override with `registry=` on the command line to use the default Docker Hub/local image (e.g. `make tools-docker registry=`).

**Apk / Alpine mirror**: The tools image uses Alpine; `apk update` runs inside the build container. DNS and network there can differ from your host. The Dockerfile retries apk up to 3 times. If it still fails, pass an alternate mirror: `make tools-image-build alpine_mirror=https://your-mirror.example.com`. Use the mirror base URL only (e.g. `https://mirror.example.com`); the path `/alpine/...` is preserved.

## CI pipeline (ci.mk, build/woodpecker.jsonnet)

- **Source of truth**: `build/woodpecker.jsonnet`. Run **`make ci`** to generate `.woodpecker.yml` and run the same steps CI runs locally (compile + test). Run **`make ci-pipeline`** to only regenerate the pipeline YAML. Jsonnet and jq run in the tools container. **Prerequisite**: build the tools image first (`make tools-image-build`).
- **Pipeline**: Woodpecker CI. All steps use the tools image from the registry. **Every run** (PR and main): compile (`make compile-only`). **Main only** (push to main): build and push the tools image, then build and push the app image. Docker-in-Docker (`docker:24-dind`) is used for image build steps.
- **First-time setup**: Push the tools image once so Woodpecker can pull it: `make tools-image-build registry=reg.dist.svc.cluster.znet:5000 && make tools-image-push registry=reg.dist.svc.cluster.znet:5000` (adjust `registry` to match `build/woodpecker.jsonnet`). Point Woodpecker at this repo and use `.woodpecker.yml` as the config path. **Kubernetes backend**: the pipeline uses DinD over the network (`DOCKER_HOST=tcp://docker:2375`), so no `dockersock` PVC is required (avoids RWX storage; works with local-path and other RWO-only storage). If the agent runs on dedicated CI nodes (e.g. `nodeSelector: workload/ci: "true"`), set **WOODPECKER_BACKEND_K8S_POD_NODE_SELECTOR** on the agent so all pipeline pods (steps and services) schedule on the same pool, e.g. `WOODPECKER_BACKEND_K8S_POD_NODE_SELECTOR: '{"workload/ci":"true"}'`. Then scaling means adding more nodes with that label.
- **Customize**: Edit `build/woodpecker.jsonnet` (locals `registry`, `toolsImage`, or add steps). Variables: `CI_CONFIG` (default `.woodpecker.yml`), `CI_JSONNET_SOURCE` (default `build/woodpecker.jsonnet`).
- **DNS**: The pipeline sets `clone.git.dns` so the clone step uses explicit nameservers. If the same failure persists or you need pod-level `dnsOptions` (ndots, timeout), see **build/woodpecker-dns-kubernetes.md** for trying the cluster DNS IP and for direct Kubernetes options (CoreDNS, MutatingAdmissionWebhook).

## Optional features

- **Protos**: Include `build/proto.mk` for projects using buf; it runs `buf build`, `buf lint`, `buf generate` in the tools container. If you don’t include `proto.mk`, the `proto` target is a no-op (from util.mk).
- **Operator/manager image**: Include `build/kube_builder.mk` for `docker-build` and `docker-push` targets. Omit for plain apps that only use `docker.mk`.
- **Commit lint**: Add a fragment that defines `lint-commit` (e.g. `build/commitlint.mk`) and include it; otherwise `lint-commit` is a no-op.

## Reference project

This repo (streamgo) is kept as a minimal example: small root Makefile, no proto, no kube_builder. Copy `build/` and adjust variables and includes for projects that need protos or operator builds.
