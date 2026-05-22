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