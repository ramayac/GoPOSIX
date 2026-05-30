# Blueprint: Go-Alpine MVP Plan

This document outlines the design blueprint for the **Go-Alpine** project—swapping out Alpine Linux's standard C-based BusyBox userland and routing it entirely through the official GoPOSIX multicall binary.

---

## 🛠️ The Optimized Single-Stage Design

Rather than attempting risky runtime symlink re-routing or high-overhead multi-stage compilations inside the container, we leverage the native symlink design of Alpine Linux:
1. **Pre-linked userland**: Alpine's tools are already symlinked to `/bin/busybox` and `/bin/sh` (which points to `/bin/busybox`).
2. **Atomic Swap**: Overwriting `/bin/busybox` with our GoPOSIX binary instantly routes all commands (including `ls`, `grep`, `pwd`, and `awk`) through GoPOSIX, without having to delete and recreate hundreds of filesystem symlinks.

### The Production Dockerfile
```dockerfile
FROM alpine:3.20

# Copy our precompiled static GoPOSIX v1.0.15 binary
# Overwriting /bin/busybox immediately re-routes all system symlinks!
COPY goposix /bin/goposix
COPY goposix /bin/busybox

# Copy our premium interactive test script
COPY test_goposix.sh /usr/local/bin/test_goposix.sh

# Start the interactive pure-Go shell environment
CMD ["/bin/sh"]
```

---

## 🚀 How to Run and Test

1. **Build the image**:
   ```bash
   docker build -t go-alpine .
   ```
2. **Boot the interactive shell**:
   ```bash
   docker run -it --rm go-alpine
   ```
3. **Run the automated diagnostic suite**:
   ```bash
   docker run --rm go-alpine /usr/local/bin/test_goposix.sh
   ```

---

## 🖼️ Graphical Openbox Desktop (Dockerfile.openbox)

GoPOSIX also supports a minimal graphical Alpine userland running the Openbox window manager. This environment is defined in `docker/Dockerfile.openbox` and demonstrates GoPOSIX's compatibility as a utility provider for standard X11 session tooling and terminal emulators.

### Design Differences
Unlike the `alpine-mvp` image which overwrites `/bin/busybox` atomically, the Openbox desktop environment:
1. **Compiles Statically**: Compiles the GoPOSIX binary using `CGO_ENABLED=0` inside `golang:1.26-alpine`.
2. **Individual Symlinks**: Lists all GoPOSIX commands via `/goposix --list-commands` and creates individual command symlinks inside `/bin/` (e.g. `/bin/ls -> /bin/goposix`).
3. **Non-Root Confinement**: Adds a `nonRootUser` and assigns the default shell to `/bin/shell` (the GoPOSIX-native shell).
4. **Window Manager Autostart**: Auto-starts `xterm` inside an Openbox session so users have immediate access to the GoPOSIX terminal shell.

### How to Run the Openbox Demo
A dedicated nested X11 Xephyr target is provided in the Makefile. 

1. **Verify Xephyr is installed on your host OS** (e.g. `xserver-xephyr` or `xorg-server-xephyr`).
2. **Launch the Demo**:
   ```bash
   make openbox-demo
   ```
   This automatically starts Xephyr on display `:1` at a 1280x720 resolution, boots the container with dynamic X11 forwarding, and spawns the Openbox session displaying an `xterm` window backed by the GoPOSIX shell.

---

## 🔧 Daemon Mode in Alpine

The current `alpine-mvp` image runs GoPOSIX as a pure CLI drop-in — it replaces
`/bin/busybox` and drops into an interactive shell (`/bin/sh`). No daemon,
no socket, no JSON-RPC. This is by design: the `alpine-mvp` target is about
testing GoPOSIX as a BusyBox replacement in a real distro.

To run GoPOSIX as a daemon inside Alpine, two changes are needed:

1.  **Entrypoint**: Replace `CMD ["/bin/sh"]` with
    `ENTRYPOINT ["/bin/goposix", "daemon", "--socket", "/home/goposix/goposix.sock", ...]`
    — same as the scratch `daemon` target.

2.  **User setup**: The daemon writes its socket to `/home/goposix/goposix.sock`
    and runs as `USER goposix` in the scratch image. Alpine ships with only
    `root` by default, so you'd need `RUN addgroup/adduser` like the `debug`
    target. Alternatively, keep root and use `/tmp/goposix.sock` to skip user
    setup entirely.

### BusyBox override: keep or drop?

Two approaches when adding the daemon to Alpine:

| Approach | BusyBox | Use Case |
|---|---|---|
| **Keep override** | Overwrite `/bin/busybox` AND start the daemon. All shell commands route through GoPOSIX. | Pure-Go experiment, full GoPOSIX userland |
| **Drop override** | Copy `goposix` only to `/bin/goposix`, leave BusyBox intact. Daemon runs alongside Alpine's native tools. | Practical use — Alpine's full ecosystem (`sh`, `apk`, init scripts) plus GoPOSIX's 60µs JSON-RPC |

The second approach is more practical for real workloads: you get Alpine's
package manager, init system, and shell scripts working normally, while
the daemon serves JSON-RPC on the socket. This is currently **not**
implemented — tracked in [todos.md](todos.md).

## 🔬 Core Verification Techniques

1. **Programmatic API Testing**:
   Check if the system supports GoPOSIX's signature `--json` output flag for automated machine parsing:
   ```bash
   ls --json /
   ```
2. **Go Executable Metadata Check**:
   Validate that `/bin/busybox` contains Go runtime signatures and has no dynamic C links:
   ```bash
   strings /bin/busybox | grep -E "go.production|Go build ID"
   ```
3. **The Package Manager Acid Test**:
   Verify that GoPOSIX's internal archive decompression and filesystem streaming are robust enough to handle production software installations:
   ```bash
   apk update && apk add --simulate curl
   ```