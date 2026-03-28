//go:build tools

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	// drone CLI is built from clone in install.sh (harness/drone-cli go.mod has replace directives; go install fails)
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/go-jsonnet/cmd/jsonnet"
	_ "github.com/google/go-jsonnet/cmd/jsonnetfmt"
	_ "github.com/grafana/tanka/cmd/tk"
	_ "github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb"
	_ "github.com/psampaz/go-mod-outdated"
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "gotest.tools/gotestsum"
	_ "mvdan.cc/gofumpt"
	_ "sigs.k8s.io/kind"
)
