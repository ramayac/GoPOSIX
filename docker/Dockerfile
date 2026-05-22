# syntax=docker/dockerfile:1

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

# Stage 3: Base environment (ca-certs, tzdata, users)
FROM scratch AS base-runtime

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV ZONEINFO=/usr/share/zoneinfo

COPY --from=symlinker /etc/passwd /etc/passwd
COPY --from=symlinker /etc/group  /etc/group
COPY --from=symlinker /out/bin/ /bin/
COPY --chown=1000:1000 --from=symlinker /home/goposix /home/goposix

USER goposix

# Target 4: CLI — production CLI-only image.
FROM base-runtime AS cli
ENTRYPOINT ["/bin/goposix"]

# Target 5: Alpine MVP Distro — Alpine base with GoPOSIX overwriting busybox.
FROM alpine:3.20 AS alpine-mvp
COPY --from=builder /goposix /bin/goposix
COPY --from=builder /goposix /bin/busybox
COPY test/compliance/test_alpine.sh /usr/local/bin/test_goposix.sh
CMD ["/bin/sh"]

# Target 6: Debug — Alpine-based debug image with utility tools.
FROM alpine:3.20 AS debug

RUN apk add --no-cache strace file

COPY --from=builder /goposix /bin/goposix
COPY --from=builder /commands.txt /commands.txt

RUN while IFS= read -r cmd; do ln -s /bin/goposix "/bin/${cmd}"; done < /commands.txt

RUN addgroup -g 1000 goposix && adduser -u 1000 -G goposix -s /bin/sh -D goposix

USER goposix
CMD ["/bin/sh"]

# Target 7: Daemon — default production daemon image (last = default).
FROM base-runtime AS daemon
EXPOSE 8080
ENTRYPOINT ["/bin/goposix", "daemon", "--socket", "/home/goposix/goposix.sock", "--workers", "4"]
CMD ["--listen-addr", ":8080"]
