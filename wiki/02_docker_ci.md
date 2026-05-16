# Phase 02 — Docker Scratch Build + CI Pipeline

> **Status:** COMPLETED / MAINTAINED | **Last verified:** 2026-05-16

---

## Goal

Package KoreGo into a `FROM scratch` Docker image with symlinks for every utility, and set up CI to prevent regressions. This phase was completed in Week 2 and has been continuously maintained through Phase 14.

---

## Docker Configuration (Current State)

### `docker/Dockerfile` — Production `FROM scratch` image

Three-stage build: **builder** → **symlinker** → **scratch**.

**Key design decisions (as-built, post-Phase 02 evolution):**

| Decision | Rationale |
|----------|-----------|
| `golang:1.26-alpine` base | ≥ go.mod `go 1.25.0`; newer compiler = better optimizations + security fixes |
| `COPY go.mod ./` then `go mod download` (no go.sum) | Layer caching — `go mod download` only needs `go.mod`. `go.sum` arrives with `COPY . .` and is used by `go build` |
| `ARG TARGETARCH` + `GOARCH=${TARGETARCH}` | Multi-arch support (amd64, arm64) via `docker buildx --platform` |
| `/out/bin/` staging directory (not `/bin/`) | **Scratch Image Purity** — avoids pulling in Alpine's BusyBox binaries. Only KoreGo + symlinks land in the final image |
| System tzdata (`/usr/share/zoneinfo`) from `apk add tzdata` | Alpine-native; more complete than Go's bundled `zoneinfo.zip` |
| `-X pkg/common.Version=... -X github.com/.../korego.Version=...` | Two version variables: library-level (`output.go`) and module-root (`korego.go`, used by `--version`) |
| `apk add ca-certificates` | HTTPS support for future utilities |
| Non-root user `korego:1000:1000` with `/home/korego` | Security best practice for `FROM scratch` |
| `ENTRYPOINT ["/bin/korego"]` | Multicall dispatch: `docker run korego ls -la` works directly |

**Symlink registry:** The builder stage runs `korego --list-commands` to emit one command name per line (55 utilities as of Phase 14c). The symlinker stage creates `/out/bin/<name> → /bin/korego` symlinks for each.

### `docker/Dockerfile.debug` — Alpine debug image

| Decision | Rationale |
|----------|-----------|
| Same `golang:1.26-alpine` builder | Identical binary to production |
| Final: `alpine:3.20` with `strace`, `file` | Full debug tooling |
| `CMD ["/bin/sh"]` (not `ENTRYPOINT`) | Allows `docker run -it korego:debug sh` without passing sh as a subcommand argument |
| Non-root user via `adduser` | Mirrors production UID/GID 1000:1000 |
| Hardcoded `GOARCH=amd64` | Debug only — no multi-arch needed |

---

## Makefile Docker Targets (Current)

```makefile
docker:        Build production scratch image (korego:$(VERSION))
docker-debug:  Build Alpine debug image (korego:debug)
docker-shell:  docker-debug + interactive shell in container
docker-run:    docker + run a command (e.g., make docker-run CMD="ls -la")
smoke-docker:  Run smoke tests inside the production scratch container
```

The `docker-run` target passes `$(CMD)` directly to the container, where the korego entrypoint dispatches it as a subcommand.

---

## GitHub Actions CI (`.github/workflows/ci.yml`)

**Current pipeline** (evolved from Phase 02 original through Phase 14):

| Step | Detail |
|------|--------|
| Go version | `1.26` (matches Dockerfile) |
| Vet | `make vet` |
| Unit tests | `make test` across all `./pkg/...` and `./internal/...` |
| Coverage gate | Hard-fail at <45% overall coverage |
| Binary build | `make build` + `--list-commands` verification |
| JSON schema validation | `make validate-schemas` |
| Docker build | `make docker` (production scratch image) |
| Smoke tests | `make smoke` in-container |
| Trivy vulnerability scan | CRITICAL/HIGH CVEs fail the build (added Phase 12.1) |
| BusyBox test suite | `make testsuite` — baseline: 477 passed, fail if < 409 |
| Image size gate | Separate job: multi-arch buildx, fails if > 20 MB |

---

## Release Pipeline (`.github/workflows/release.yml`)

Tagged releases (`v*`) trigger GoReleaser, which uses `docker/Dockerfile` with buildx to produce:

- Per-arch images: `ghcr.io/ramayac/korego:$(VERSION)-amd64`, `-arm64`
- Multi-arch manifest: `ghcr.io/ramayac/korego:$(VERSION)`, `:latest`
- SBOMs on archive and binary artifacts
- Cosign keyless signing (OIDC)
- SLSA Level 3 provenance attestation

---

## Milestone 02 (Verified Current)

- [x] `docker build` produces a working `FROM scratch` image
- [x] `docker run korego true` exits 0, `false` exits 1
- [x] Symlink dispatch works: `docker run --entrypoint /bin/echo korego hello`
- [x] `docker run korego --help` lists all 55 commands
- [x] Image size < 20 MB (actual: ~15 MB with buildx, varying by arch)
- [x] Multi-arch: linux/amd64 + linux/arm64
- [x] CI pipeline: lint → test → docker → smoke → trivy → BusyBox
- [x] `make smoke-docker` passes all checks
- [x] Supply chain: SBOM + Cosign + SLSA Level 3 on releases

## How to Verify (Current)

```bash
# Local build + test
make docker
make smoke-docker

# Run specific utility in production container
make docker-run CMD="ls -la /bin"

# Interactive debug shell
make docker-shell

# Full CI pipeline
make ci

# Image size
docker image inspect korego:dev --format '{{.Size}}'
```
