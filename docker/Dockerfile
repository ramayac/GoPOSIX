# =============================================================================
# Stage 1: Builder
# Compiles a fully static binary with no CGO.
# =============================================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ARG VERSION
ARG TARGETARCH
RUN VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")} && \
    TARGETARCH=${TARGETARCH:-amd64} && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build \
      -ldflags="-s -w -X 'github.com/ramayac/korego/pkg/common.Version=${VERSION}' -X 'github.com/ramayac/korego.Version=${VERSION}'" \
      -o /korego ./cmd/korego/

# Emit one command name per line for the symlink stage.
RUN /korego --list-commands > /commands.txt

# Stage 2: Symlink generator
# =============================================================================
FROM alpine:3.20 AS symlinker

# We create a clean directory for our binary and symlinks to avoid 
# pulling in alpine's busybox and other binaries.
RUN mkdir -p /out/bin

COPY --from=builder /korego /out/bin/korego
COPY --from=builder /commands.txt /commands.txt

# Create /bin symlinks for every registered command inside /out/bin
RUN while IFS= read -r cmd; do ln -s /bin/korego "/out/bin/${cmd}"; done < /commands.txt

# Non-root user & group for the scratch stage.
RUN echo "korego:x:1000:1000::/home/korego:/bin/false" >> /etc/passwd && \
    echo "korego:x:1000:" >> /etc/group && \
    mkdir -p /home/korego && chown 1000:1000 /home/korego

# =============================================================================
# Stage 3: Scratch — final minimal image
# =============================================================================
FROM scratch

# CA certificates (needed for future HTTPS utilities).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Timezone data (needed for the date utility).
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV ZONEINFO=/usr/share/zoneinfo

# Non-root user.
COPY --from=symlinker /etc/passwd /etc/passwd
COPY --from=symlinker /etc/group  /etc/group
COPY --from=symlinker /home/korego /home/korego

# Copy our clean /bin directory.
COPY --from=symlinker /out/bin/ /bin/

USER korego
ENTRYPOINT ["/bin/korego"]
