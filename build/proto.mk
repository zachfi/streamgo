#
# build/proto.mk - Protobuf code generation (runs in tools container)
#
# Uses buf (buf build, buf lint, buf generate) via TOOLS_CMD so no protoc/buf
# need to be installed on the host. Include this only in projects that use protobuf.
#
# Prerequisite: tools image (make tools-image-build). Configure buf in the repo
# (e.g. buf.yaml, buf.gen.yaml) as usual.
#

proto: proto-grpc gofmt-fix

proto-grpc:
	@echo "=== $(PROJECT_NAME) === [ proto            ]: compiling protobufs (buf)..."
	@$(TOOLS_CMD) buf build
	@$(TOOLS_CMD) buf lint
	@$(TOOLS_CMD) buf generate

.PHONY: proto proto-grpc
