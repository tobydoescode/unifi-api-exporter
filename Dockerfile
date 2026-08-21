# syntax=docker/dockerfile:1.26
# Multi-arch via BUILDPLATFORM; Go cross-compiles to $TARGETARCH (no QEMU needed).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY main.go ./
COPY internal/ ./internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/unifi-api-exporter ./

FROM gcr.io/distroless/static-debian13:nonroot@sha256:1c2c046bc09ed40fad370b599a0b1ae7987f55b01e247cf27a7c27cd97e5bbc7
LABEL org.opencontainers.image.source="https://github.com/tobydoescode/unifi-api-exporter" \
      org.opencontainers.image.description="Prometheus exporter for UniFi device state" \
      org.opencontainers.image.licenses="MIT"
COPY --from=build /out/unifi-api-exporter /unifi-api-exporter
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/unifi-api-exporter"]
