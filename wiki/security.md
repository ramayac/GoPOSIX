# Security Model

## Trust Level

The `goposix.shell.exec` RPC method is designed for **trusted input only**. Scripts executed
through this interface run with the same OS-level privileges as the daemon process. The
sandboxing below is a defense-in-depth measure, not a guarantee against a determined
adversary with arbitrary code execution.

If you expose the daemon socket to untrusted clients (e.g., over a network), you must
implement authentication and authorization in front of it — the daemon itself has no
auth layer.

## Accessible Resources

### Filesystem

Scripts can access the filesystem subject to path confinement via `SecurePath`:

- **Symlinks limitation:** `SecurePath` currently uses lexical path cleaning (`filepath.Clean`) and does *not* resolve symlinks at the application level (i.e. it does not call `filepath.EvalSymlinks`). If a symlink in the base directory points to a location outside the base directory, access to the external path may be possible.

> [!WARNING]
> **CWD Path Confinement Limitation:** The `session.setCwd` RPC method does not currently validate that the target directory is restricted to a safe subtree before setting it as the session's working directory. Subsequent commands in that session will then use that new directory as their base path for confinement.

### Environment Variables

Scripts inherit a caller-specified environment. By default, no environment variables
are set. Callers can inject specific variables via the `env` parameter.

### Network

**No network access.** The shell interpreter (`mvdan.cc/sh`) is configured without
networking capabilities. Scripts cannot make outbound connections or listen on ports.

### Process Execution

Scripts can invoke GoPOSIX utilities (the same ones available via the multicall binary)
and any external commands available in the container/VM. On a `FROM scratch` Docker
image, only GoPOSIX utilities are available.

## Resource Limits

| Resource | Limit | Configurable |
|----------|-------|-------------|
| Execution timeout | 30s default | `GOPOSIX_SHELL_TIMEOUT` env var (Go duration format, e.g. `60s`, `5m`) |
| Stdout buffer | 128 MB | No (hardcoded `LimitWriter`) |
| Stderr buffer | 128 MB | No (hardcoded `LimitWriter`) |

When the timeout expires, the script is terminated via context cancellation and receives
a non-zero exit code. When either output buffer is exceeded, the stream is truncated.

## RPC-Level Protections

| Protection | Value |
|-----------|-------|
| Max request body size | 1 MB |
| Max RPC requests/sec per connection | 100,000 (effectively unrestricted) |
| Connection limit | Configurable max concurrent connections |
| Session TTL | Automatic cleanup of idle sessions |

## Recommended Deployment Posture

1. **Run as non-root.** The Docker image uses `USER goposix (1000:1000)`. Never override
   this without a specific reason.
2. **Socket permissions.** The Unix socket should be owned by the daemon user with `0600`
   or `0660` permissions. Restrict access to a specific group.
3. **No network exposure.** Use Unix domain sockets, not TCP. If TCP is unavoidable, place
   a reverse proxy with authentication in front of the daemon.
4. **Minimal base image.** Use the `FROM scratch` production image. It contains only the
   GoPOSIX static binary — no shell, no package manager, no utilities.
5. **Read-only filesystem.** Mount the container filesystem as read-only except for the
   socket directory and any paths the daemon needs to write to.
6. **Session isolation.** Use sessions (`goposix.session.create`) for multi-step workflows.
   Sessions confine file operations to a working directory and carry environment state
   across calls. Stateless calls operate against `/` by default.
   > [!CAUTION]
   > Note that `session.setCwd` does not validate path boundaries. Do not permit untrusted clients to invoke `session.setCwd` with arbitrary path inputs.

### Should `shell.exec` require a session?

**Decision: No.** Sessions are optional for `shell.exec` (sessionId can be empty).

**Rationale:** Stateless one-off shell commands are a valid and common use case (e.g., `echo hello`)
that should not require session-creation boilerplate. The security boundaries — timeout enforcement,
output buffer limits, `SecurePath` confinement — apply identically with or without a session.
Sessions provide CWD context and environment state for multi-step workflows, not a security boundary.
If a deployment requires stronger isolation, restrict access at the socket permission level.

## Artifact Verification

Release artifacts are signed via Cosign keyless signing (OIDC) and include SLSA Level 3
provenance attestations. SBOMs are attached to container images and binary archives.
CI scans the Docker image with Trivy on every push for CRITICAL/HIGH CVEs.

```bash
# Verify a release image is signed by this repo's GitHub Actions
cosign verify ghcr.io/ramayac/goposix:latest \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'

# Inspect the SBOM attached to the image
docker buildx imagetools inspect ghcr.io/ramayac/goposix:latest --format '{{ json .SBOM }}'

# Verify SLSA provenance (requires slsa-verifier)
slsa-verifier verify-image ghcr.io/ramayac/goposix:latest \
  --source-uri github.com/ramayac/goposix
```

## Reporting Vulnerabilities

Please report security issues via GitHub's private vulnerability reporting on the
repository. Do not open public issues for security bugs.
