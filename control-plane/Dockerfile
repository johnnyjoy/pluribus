# Control-plane API only (Pluribus HTTP-first runtime).
# CI publishes multi-arch to GHCR: ghcr.io/<github-owner-lowercase>/pluribus — platforms match postgres:18-alpine ∩ redis:7 (see docs/pluribus-image-release-policy.md)
# Local build (version embedded in binary + image tags): from repo root, `make image`
# Manual: docker build -t pluribus:local --build-arg VERSION=$(git describe --tags --always --dirty) -f control-plane/Dockerfile control-plane
# Run:  docker run --rm -p 8123:8123 -e CONFIG=/config/config.yaml pluribus:local
# Compose: docker-compose.yml (build from source) or docker-compose.install.yml (set PLURIBUS_IMAGE to a local tag after `make image`).

FROM golang:1.22-alpine AS build
ARG VERSION=dev
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/controlplane ./cmd/controlplane

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /
COPY --from=build /out/controlplane /usr/local/bin/controlplane
COPY scripts/entrypoint /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/entrypoint
COPY configs/config.yaml /config/config.yaml
ENV CONFIG=/config/config.yaml
EXPOSE 8123
ENTRYPOINT ["/usr/local/bin/entrypoint"]
