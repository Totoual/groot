# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.4

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build \
      -trimpath \
      -ldflags="-s -w -X github.com/totoual/groot/internal/versioninfo.ReleaseVersion=${VERSION}" \
      -o /out/groot \
      ./cmd/groot

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 10001 groot \
    && useradd --uid 10001 --gid groot --home-dir /var/lib/groot --shell /bin/sh groot \
    && mkdir -p /var/lib/groot /workspace \
    && chown -R groot:groot /var/lib/groot /workspace

COPY --from=build /out/groot /usr/local/bin/groot

ENV GROOT_HOME=/var/lib/groot
WORKDIR /workspace

USER groot
ENTRYPOINT ["groot"]
CMD ["mcp", "--http", "--listen", "0.0.0.0:8080", "--endpoint", "/mcp", "--project", "/workspace"]
