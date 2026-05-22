# 🏔️ Go-Alpine (GoPOSIX-powered Alpine Linux)

Welcome to **Go-Alpine**! This is a bleeding-edge, highly optimized Linux container userland that replaces Alpine's traditional C-based BusyBox system with the **official GoPOSIX (v1.0.15)** release—a 100% pure, statically compiled, Go-native multicall environment.

---

## 🛠️ Architecture & How It Works

Alpine Linux is exceptionally light because almost all core command-line tools (`/bin/sh`, `/bin/ls`, `/bin/cp`, `/bin/grep`, etc.) are symbolic links pointing directly to a single multicall binary: `/bin/busybox`.

By compiling the custom GoPOSIX binary statically (`CGO_ENABLED=0`) and replacing `/bin/busybox` directly, we achieve a **complete, atomic swap** of the OS userland:
1. **Zero Symlink Disruption**: Existing system symlinks are preserved and now seamlessly invoke GoPOSIX.
2. **Upstream `sh` Integration**: Starting in GoPOSIX **v1.0.15**, the `"sh"` shell is registered natively in the command dispatcher, allowing the container to boot and run complex shell scripts/shebangs out-of-the-box.
3. **Symlink Safety**: The `"sh"` command is automatically excluded from GoPOSIX's `--list-commands` flag, ensuring automated local environment symlink tools remain safe.

---

## 🚀 Getting Started (Build & Run)

Our Docker build uses a highly optimized, multi-stage build. You can build the Alpine Distro image directly using:

### 1. Build the Image
```bash
# Build the Alpine Distro image (compiles the binary internally)
docker build --target alpine-mvp -t go-alpine -f docker/Dockerfile .
```

### 2. Boot into the Pure Go Userland
To run the interactive Go shell environment inside your new container:
```bash
docker run -it --rm go-alpine
```

---

## 🧪 Verifying the Environment

Once inside the container, you can check that you are running on GoPOSIX using three methods:

### 1. Run the Diagnostic Test Suite
We have pre-loaded a premium interactive testing script inside the image. To run it:
```bash
# From inside the container:
/usr/local/bin/test_goposix.sh

# Or directly from the host shell:
docker run --rm go-alpine /usr/local/bin/test_goposix.sh
```

### 2. Programmatic Machine-Readable JSON Output (GoPOSIX Signature)
GoPOSIX utilities are designed for programmatic consumption. Every tool supports a custom `--json` flag to print structured machine-readable JSON output instead of plain text:
```bash
# Output directory listings as JSON
ls --json /
```
**Example Output:**
```json
{
  "command": "ls",
  "version": "1.0.15",
  "schemaVersion": "1.0",
  "exitCode": 0,
  "data": {
    "path": "/",
    "files": [
      {"name": "bin", "size": 14, "mode": "drwxr-xr-x", "isDir": true, "owner": "root"},
      {"name": "etc", "size": 56, "mode": "drwxr-xr-x", "isDir": true, "owner": "root"}
    ]
  },
  "error": null
}
```

### 3. Engine Version Check
Query the version directly from the multicall system:
```bash
busybox --version
```
*Expected Output:* `goposix version 1.0.15`

---

## ⚔️ The Acid Tests

To prove the robustness of GoPOSIX as a replacement for standard BusyBox, you can execute these advanced tests:

### 📦 Live Package Management
GoPOSIX tar and gzip utilities are fully compatible with production archive parsing. You can execute Alpine's `apk` package manager to update indices and simulate real package downloads:
```bash
# From inside the container:
apk update && apk add --simulate curl
```

### 🐉 Play NetHack!
Because GoPOSIX's interactive shell fully implements standard terminal bindings and character streams, you can run complex curses/terminal-based games. Let's install and play NetHack:
```bash
# 1. Run the container with interactive terminal flags:
docker run -it --rm go-alpine

# 2. Inside the shell, add the package and play:
apk add nethack
nethack
```

---

## 📄 License & Contributing
This project is open-source. Feel free to clone, open PRs, and explore the future of Go-native operating systems!
