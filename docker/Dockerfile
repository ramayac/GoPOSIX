# =============================================================================
# GoPOSIX default image — persistent JSON-RPC 2.0 daemon.
#
# This is the recommended production image. The daemon starts automatically
# and listens on /var/run/goposix.sock for Go SDK or JSON-RPC clients.
#
# Usage:
#   docker build -t goposix:latest .
#   docker run -d --name goposix goposix:latest
#
#   # Connect via Go SDK (60µs per call, 11× faster than BusyBox fork+exec):
#   c, _ := client.New("/var/run/goposix.sock")
#   c.Echo(ctx, "hello")
#
#   # For CLI-only access, use goposix:cli instead:
#   docker pull ghcr.io/ramayac/goposix:cli
# =============================================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION
RUN VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")} && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-s -w -X 'github.com/ramayac/goposix/pkg/common.Version=${VERSION}' -X 'github.com/ramayac/goposix.Version=${VERSION}'" \
      -o /goposix ./cmd/goposix/

RUN /goposix --list-commands > /commands.txt

# =============================================================================
# Final: Alpine base with daemon. Symlinks provide docker exec CLI access.
# =============================================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata strace file

COPY --from=builder /goposix /bin/goposix
COPY --from=builder /commands.txt /commands.txt

# Symlinks for direct CLI access (docker exec goposix ls -la /).
RUN while IFS= read -r cmd; do ln -s /bin/goposix "/bin/${cmd}"; done < /commands.txt

# Non-root user.
RUN addgroup -g 1000 goposix && adduser -u 1000 -G goposix -s /bin/sh -D goposix
RUN mkdir -p /var/run && chown goposix:goposix /var/run

USER goposix
EXPOSE 8080

# Start the daemon on a Unix socket (and optional HTTP for remote access).
ENTRYPOINT ["/bin/goposix", "daemon", "--socket", "/var/run/goposix.sock", "--workers", "4"]
CMD ["--listen-addr", ":8080"]
