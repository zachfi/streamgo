# Match tools/go.mod and main app (Go 1.26).
FROM golang:1.26-alpine

ARG TARGETOS
ARG TARGETARCH
# Override Alpine mirror if container DNS/network differs from host (e.g. make tools-image-build alpine_mirror=https://mirror.example.com)
ARG ALPINE_MIRROR=
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
# Ensure go-installed tools (drone, golangci-lint, buf, etc.) are on PATH
ENV PATH=/go/bin:$PATH

RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s|https*://dl-cdn.alpinelinux.org|$ALPINE_MIRROR|g" /etc/apk/repositories; fi; \
	for i in 1 2 3; do apk update && apk add --no-cache make git bash protoc jq && { rm -rf /var/cache/apk/*; exit 0; }; sleep 2; done; exit 1
# docker-cli (edge/community, API 1.44+) and drone-cli (edge/testing); add edge repos, install, then remove so we don't mix edge with base
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
	echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && \
	apk update && apk add --no-cache docker-cli drone-cli && \
	sed -i '/edge\/community/d' /etc/apk/repositories && sed -i '/edge\/testing/d' /etc/apk/repositories && rm -rf /var/cache/apk/*

WORKDIR /tools
COPY tools /tools
# Populate module cache so go install pkg@version can resolve transitive deps (e.g. cel.dev/expr -> go-cmp@v0.6.0).
RUN go mod download && go mod download github.com/google/go-cmp@v0.6.0
RUN /tools/install.sh && \
	go clean -cache && go clean -modcache && \
	rm -rf /tools

COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
