# syntax=docker/dockerfile:1

# =============================================================================
# GoPOSIX daemon — persistent JSON-RPC server (default production image).
# Statically linked, FROM scratch, ~10 MB.
#
# This is the recommended production image. The daemon starts automatically
# and listens on /home/goposix/goposix.sock for Go SDK or JSON-RPC clients.
#
# Usage:
#   docker build -t goposix:latest -f docker/Dockerfile .
#   docker run -d --name goposix -p 8080:8080 goposix:latest
#
#   # Connect via Go SDK (60µs per call, 11× faster than BusyBox fork+exec):
#   c, _ := client.New("/home/goposix/goposix.sock")
#   c.Echo(ctx, "hello")
#
#   # For CLI-only access, use goposix:cli instead:
#   docker pull ghcr.io/ramayac/goposix:cli
# =============================================================================

# Stage 1: Builder — compiles a fully static binary.
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION
ARG TARGETARCH
RUN VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")} && \
    TARGETARCH=${TARGETARCH:-amd64} && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build \
      -ldflags="-s -w -X 'github.com/ramayac/goposix/pkg/common.Version=${VERSION}' -X 'github.com/ramayac/goposix.Version=${VERSION}'" \
      -o /goposix ./cmd/goposix/

RUN /goposix --list-commands > /commands.txt

# Stage 2: Symlink generator — creates /out/bin with binary + symlinks.
FROM alpine:3.20 AS symlinker

RUN mkdir -p /out/bin

COPY --from=builder /goposix /out/bin/goposix
COPY --from=builder /commands.txt /commands.txt

RUN while IFS= read -r cmd; do ln -s /bin/goposix "/out/bin/${cmd}"; done < /commands.txt

# User/group files and home directory for non-root execution.
RUN echo "goposix:x:1000:1000::/home/goposix:/bin/false" >> /etc/passwd && \
    echo "goposix:x:1000:" >> /etc/group && \
    mkdir -p /home/goposix && \
    chown 1000:1000 /home/goposix

# Stage 3: Scratch — minimal production image.
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV ZONEINFO=/usr/share/zoneinfo

COPY --from=symlinker /etc/passwd /etc/passwd
COPY --from=symlinker /etc/group  /etc/group
COPY --from=symlinker /out/bin/ /bin/

# --chown preserves ownership for both files and directories in FROM scratch.
COPY --chown=1000:1000 --from=symlinker /home/goposix /home/goposix

USER goposix
EXPOSE 8080

# Start the daemon on a Unix socket (and optional HTTP for remote access).
ENTRYPOINT ["/bin/goposix", "daemon", "--socket", "/home/goposix/goposix.sock", "--workers", "4"]
CMD ["--listen-addr", ":8080"]
