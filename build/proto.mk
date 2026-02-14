#
# build/proto.mk - Protobuf/GRPC code generation
#
# Include this only in projects that use protobuf. In your Makefile, set
# PROTO_GRPC_FILES to the list of .proto files (e.g. file.proto or dir/file.proto).
# Optional: PROTO_IMPORT_DIRS for -I flags (default: .)
#
# Example (in your Makefile):
#   PROTO_GRPC_FILES = rpc/rpc.proto pkg/foo/foo.proto
#   include build/vars.mk
#   include build/proto.mk
#
PROTO_IMPORT_DIRS ?= .
PROTO_GRPC_FILES  ?=

proto: proto-grpc gofmt-fix

proto-grpc:
ifneq ($(PROTO_GRPC_FILES),)
	@echo "=== $(PROJECT_NAME) === [ proto compile    ]: compiling protobufs"
	@protoc $(addprefix -I ,$(PROTO_IMPORT_DIRS)) \
		--go_out=./ --go_opt=paths=source_relative \
		--go-grpc_out=./ --go-grpc_opt=paths=source_relative \
		$(PROTO_GRPC_FILES)
else
	@echo "=== $(PROJECT_NAME) === [ proto            ]: PROTO_GRPC_FILES not set, skipping"
endif

.PHONY: proto proto-grpc
