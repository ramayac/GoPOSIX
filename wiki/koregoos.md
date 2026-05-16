# KoreGoOS — Bootable Linux Distro (powered by KoreGo)

> ⚠️ **MOVED:** This document has been relocated to the [KoreGoOS repository](https://github.com/ramayac/koregoos).
> The copy here is retained as a historical snapshot and will not be updated.
> For the current version, see `docs/design.md` in the KoreGoOS repo.
>
> **Status:** MOVED | **Original Date:** 2026-05-16 | **Depends on:** KoreGo v1.0 (frozen userland)

---

## Concept

KoreGoOS is a minimal, bootable Linux distribution where the **entire userland** is the KoreGo multicall binary plus a small set of boot/system utilities. It imports KoreGo as a Go module and extends it through the same `dispatch.Register()` / `korego.Main()` API already exposed at `github.com/ramayac/korego`.

The result is a `FROM scratch`-style OS image that boots directly in QEMU with no libc, no package manager, no init system beyond what KoreGoOS provides — just a Linux kernel, an initramfs, and the KoreGo + KoreGoOS multicall binary.

---

## Architecture

```
┌──────────────────────────────────────────────────┐
│  KoreGoOS                                        │
│                                                  │
│  ┌──────────────────────────────────────────┐    │
│  │  Boot layer (this project)               │    │
│  │                                          │    │
│  │  init       PID 1: mounts, runs /etc/rc, │    │
│  │             spawns getty, reaps orphans  │    │
│  │  mount      mount/umount syscall wrappers │    │
│  │  mknod      device node creation          │    │
│  │  reboot     reboot/poweroff/halt          │    │
│  │  getty      terminal on /dev/console      │    │
│  │  ifconfig   network interface setup       │    │
│  │  dhclient   DHCP client                   │    │
│  │  route      routing table                 │    │
│  │  dmesg      kernel log reader             │    │
│  │  modprobe   kernel module loader          │    │
│  │  login      user authentication           │    │
│  │  passwd     password management           │    │
│  │  syslogd    system logger                 │    │
│  │  ping       network testing               │    │
│  │                                          │    │
│  │  Build tools:                             │    │
│  │  initramfs packer, kernel config,         │    │
│  │  QEMU test harness, release images        │    │
│  ├──────────────────────────────────────────┤    │
│  │  KoreGo (imported Go module)             │    │
│  │                                          │    │
│  │  Filesystem: ls, cat, cp, mv, rm,        │    │
│  │    mkdir, rmdir, touch, ln, stat,        │    │
│  │    readlink, chmod, chown, chgrp          │    │
│  │  Text: grep, sed, sort, uniq, wc,        │    │
│  │    head, tail, cut, tr, tee, diff         │    │
│  │  System: ps, kill, sleep, date, id,      │    │
│  │    hostname, uname, pwd, whoami, env,     │    │
│  │    printenv, df, du, find, xargs          │    │
│  │  Data: tar, gzip, sha256sum, md5sum       │    │
│  │  Shell: mvdan.cc/sh interpreter           │    │
│  │  Daemon: JSON-RPC 2.0 over Unix socket    │    │
│  │  Output: --json on all 56 utilities       │    │
│  └──────────────────────────────────────────┘    │
│                                                  │
│  Kernel: linux-6.x (external — user provides)     │
└──────────────────────────────────────────────────┘
```

---

## API Integration

KoreGoOS composes KoreGo through the public API (`korego.Main()` / `korego.Run()`):

```go
// cmd/koregoos/main.go
package main

import (
    "os"

    "github.com/ramayac/korego"

    // KoreGo's 56 standard utilities
    _ "github.com/ramayac/korego/pkg/ls"
    _ "github.com/ramayac/korego/pkg/cat"
    // ... all 56 blank imports

    // KoreGoOS's boot/system utilities
    _ "github.com/ramayac/koregoos/pkg/init"
    _ "github.com/ramayac/koregoos/pkg/mount"
    _ "github.com/ramayac/koregoos/pkg/umount"
    _ "github.com/ramayac/koregoos/pkg/mknod"
    _ "github.com/ramayac/koregoos/pkg/reboot"
    _ "github.com/ramayac/koregoos/pkg/poweroff"
    _ "github.com/ramayac/koregoos/pkg/halt"
    _ "github.com/ramayac/koregoos/pkg/getty"
    _ "github.com/ramayac/koregoos/pkg/ifconfig"
    _ "github.com/ramayac/koregoos/pkg/dhclient"
    _ "github.com/ramayac/koregoos/pkg/route"
    _ "github.com/ramayac/koregoos/pkg/dmesg"
    _ "github.com/ramayac/koregoos/pkg/modprobe"
    _ "github.com/ramayac/koregoos/pkg/login"
    _ "github.com/ramayac/koregoos/pkg/passwd"
    _ "github.com/ramayac/koregoos/pkg/syslogd"
    _ "github.com/ramayac/koregoos/pkg/ping"
)

func main() {
    korego.WellKnownNames = append(korego.WellKnownNames, "koregoos")
    os.Exit(korego.Main())
}
```

**Key points:**
- `korego.WellKnownNames` is extended so `koregoos ls` works (subcommand dispatch)
- All KoreGo utilities register via their `init()` functions (blank imports)
- All KoreGoOS utilities register via their `init()` functions (blank imports)
- The resulting binary is a single static ELF: `/bin/koregoos` + symlinks

---

## Boot Process

```
Kernel loads → executes /init (→ /bin/koregoos symlink)
                         │
                         ▼
              ┌─────────────────────┐
              │  koregoos init       │
              │                     │
              │  1. mount /proc     │
              │  2. mount /sys      │
              │  3. mount /dev      │  (devtmpfs or mknod)
              │  4. hostname koregoos│
              │  5. run /etc/rc     │  (korego shell interpreter)
              │  6. spawn getty on  │
              │     /dev/console    │
              │  7. signal loop     │  (reap orphans, handle
              │                     │   SIGINT/SIGTERM → shutdown)
              └─────────────────────┘
```

`/etc/rc` is a shell script executed by KoreGo's own shell interpreter (`internal/shell`):

```sh
#!/bin/koregoos shell

# Mount virtual filesystems
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

# Set hostname from config
if [ -f /etc/hostname ]; then
    hostname "$(cat /etc/hostname)"
else
    hostname koregoos
fi

# Bring up loopback
ifconfig lo 127.0.0.1 up

# Start network (optional)
if [ -f /etc/network.conf ]; then
    dhclient eth0 &
fi

# Start system logger
syslogd

# Start the JSON-RPC daemon for agent access
koregoos daemon --socket /run/korego.sock &

echo "KoreGoOS $(uname -r) ready."
```

---

## Utilities to Build

### Tier 1 — Boot-critical (must have, ~400 lines of Go)

These are non-negotiable. Without them, the kernel panics after starting `/init`.

| Utility | Purpose | Syscalls / stdlib | Est. LOC |
|---------|---------|-------------------|----------|
| `init` | PID 1: mount /proc/sys/dev, run /etc/rc, spawn getty, reap zombies, handle signals, shutdown | `unix.Mount`, `unix.Reboot`, `os.StartProcess`, signal handling | 200 |
| `mount` | Mount filesystems (CLI: `mount -t fstype dev dir`) | `unix.Mount` | 50 |
| `umount` | Unmount filesystems | `unix.Unmount` | 30 |
| `mknod` | Create device nodes (fallback if no devtmpfs) | `unix.Mknod` | 30 |
| `reboot` | Reboot the system | `unix.Reboot(LINUX_REBOOT_CMD_RESTART)` | 15 |
| `poweroff` | Power off the system | `unix.Reboot(LINUX_REBOOT_CMD_POWER_OFF)` | 15 |
| `halt` | Halt the system | `unix.Reboot(LINUX_REBOOT_CMD_HALT)` | 15 |

### Tier 2 — Usable system (~600 lines of Go)

| Utility | Purpose | Est. LOC |
|---------|---------|----------|
| `getty` | Spawn shell on /dev/console (or tty) with login | 60 |
| `login` | Authenticate user, set uid/gid, exec shell | 80 |
| `passwd` | Read/write /etc/passwd and /etc/shadow | 80 |
| `ifconfig` | Configure network interfaces (set IP, netmask, up/down) | 100 |
| `route` | View/manipulate routing table | 60 |
| `dhclient` | Basic DHCP client (UDP discovery, request, ack) | 120 |
| `ping` | ICMP echo request/reply | 60 |
| `dmesg` | Read /dev/kmsg or syslog | 40 |

### Tier 3 — "Real distro" (~800 lines of Go)

| Utility | Purpose | Est. LOC |
|---------|---------|----------|
| `modprobe` | Load kernel modules via finit_module | 60 |
| `insmod` | Load a single kernel module | 30 |
| `rmmod` | Remove a kernel module | 20 |
| `lsmod` | List loaded modules (/proc/modules) | 30 |
| `syslogd` | Simple syslog daemon (listen on /dev/log, write to /var/log/) | 120 |
| `klogd` | Kernel log daemon (read /proc/kmsg → syslog) | 60 |
| `crond` | Minimal cron daemon (read /etc/crontab, sleep-loop) | 100 |
| `fsck` | Basic filesystem check (read-only pass for ext2/btrfs) | 150 |
| `mkswap` | Create swap signature | 30 |
| `swapon/swapoff` | Enable/disable swap | 40 |
| `fdisk` | Basic partition table reader | 100 |
| `mkfs.ext2` | Minimal ext2 creation | 120 |
| `losetup` | Associate loop devices with files | 40 |
| `blkid` | Print block device attributes | 40 |

---

## Build Pipeline

```
┌────────────┐    ┌─────────────┐    ┌──────────────┐    ┌──────────┐
│ Build      │    │ Create      │    │ Pack         │    │ Boot in  │
│ koregoos   │───▶│ initramfs   │───▶│ initramfs    │───▶│ QEMU     │
│ binary     │    │ tree        │    │ .cpio.gz     │    │          │
└────────────┘    └─────────────┘    └──────────────┘    └──────────┘
```

### Step 1: Build the multicall binary

```bash
CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w -X github.com/ramayac/koregoos.Version=$(VERSION)" \
  -o koregoos ./cmd/koregoos/
```

### Step 2: Create initramfs tree

```bash
mkdir -p initramfs/{bin,dev,etc,proc,sys,run,tmp,var/log}
cp koregoos initramfs/bin/
ln -s /bin/koregoos initramfs/init           # kernel entry point
cd initramfs/bin
for cmd in $(./koregoos --list-commands); do
    ln -s /bin/koregoos "$cmd"
done
cd ../..
cp etc/rc initramfs/etc/rc
cp etc/hostname initramfs/etc/hostname
cp etc/passwd initramfs/etc/passwd
```

### Step 3: Pack into cpio

```bash
cd initramfs && find . | cpio -o -H newc | gzip > ../initramfs.cpio.gz
```

### Step 4: Boot in QEMU

```bash
qemu-system-x86_64 \
  -kernel /path/to/bzImage \
  -initrd initramfs.cpio.gz \
  -append "console=ttyS0 quiet" \
  -nographic
```

---

## Test Strategy

Different from KoreGo's unit tests. KoreGoOS needs:

| Test type | What it verifies |
|-----------|-----------------|
| **QEMU boot smoke test** | Kernel boots, init runs, /etc/rc completes, getty spawns, `exit 0` from QEMU |
| **Utility smoke in VM** | `koregoos mount` works, `koregoos reboot` triggers shutdown |
| **KoreGo regression gate** | All 477 BusyBox tests still pass when run inside KoreGoOS (korego is imported, must not regress) |
| **Init reaper test** | Zombie processes are properly reaped by PID 1 |
| **Signal handling** | SIGINT → clean shutdown, SIGTERM → poweroff, orphan processes adopted |
| **Network smoke** | DHCP lease acquired, ping loopback works |
| **Security test** | Non-root user cannot mount, cannot reboot, cannot write to /dev/mem |

---

## Repository Structure

```
github.com/ramayac/koregoos/
├── cmd/
│   └── koregoos/
│       └── main.go              # blank imports + korego.Main()
├── pkg/
│   ├── init/                    # PID 1
│   ├── mount/                   # mount + umount
│   ├── mknod/
│   ├── reboot/                  # reboot + poweroff + halt
│   ├── getty/
│   ├── login/
│   ├── passwd/
│   ├── ifconfig/
│   ├── route/
│   ├── dhclient/
│   ├── ping/
│   ├── dmesg/
│   ├── modprobe/                # modprobe + insmod + rmmod + lsmod
│   ├── syslogd/
│   ├── crond/
│   └── fsck/
├── internal/
│   └── boot/                    # shared boot helpers (reaper, console)
├── etc/                         # default /etc files (rc, passwd, hostname)
├── kernel/
│   └── config.minimal           # minimal kernel .config for QEMU
├── test/
│   ├── qemu/                    # QEMU boot test harness
│   └── smoke/                   # in-VM utility tests
├── Makefile
├── go.mod                       # require github.com/ramayac/korego v1.x
└── README.md
```

---

## Design Decisions

### Why a separate repo instead of adding to KoreGo?

| Reason | Detail |
|--------|--------|
| **Layer boundary** | KoreGo is a userland library. Booting is a different concern. |
| **Independent cadence** | KoreGo can ship v1.1 bugfix without rebuilding initramfs images. |
| **Different test profile** | KoreGo tests are `go test ./...`. KoreGoOS tests are QEMU boot smoke + kernel panic detection. |
| **Different audience** | KoreGo's consumer is an AI agent that wants `--json` output. KoreGoOS's consumer boots a minimal Linux. |
| **Clean imports** | KoreGo is a Go module dependency. No circular coupling. |

### Why use KoreGo's shell instead of bash?

KoreGo's shell interpreter (`internal/shell`, backed by `mvdan.cc/sh/v3`) already provides:
- Resource limits (timeout, 128MB stdout/stderr cap)
- Path confinement (SecurePath — no `../../../etc/passwd` escapes)
- Session isolation (cwd, env per session)
- Same binary — no additional dependency

The `/etc/rc` script runs interpreted, not exec'd. This means resource limits are enforced during boot.

> **Design observation (2026-05-16):** `internal/shell` is a library, not a registered CLI command.
> KoreGo needs a `pkg/shell/` wrapper (see `wiki/prepare_to_goose.md` Change 1) so that
> `koregoos shell /etc/rc` works. KoreGoOS cannot import `internal/` packages directly
> (Go enforces module boundaries on `internal/`). The shell must be exposed as a dispatch-registered
> command.

### Why devtmpfs instead of static /dev?

The kernel's `CONFIG_DEVTMPFS` eliminates the need for hundreds of `mknod` calls. `mknod` is included as a fallback for kernels without devtmpfs support.

### Why no package manager?

KoreGoOS is a **static** distro. The entire userland is compiled into one binary. There are no packages to manage. Updates mean rebuilding the initramfs with a new binary. This is the same model as Docker `FROM scratch` images — the build pipeline IS the package manager.

---

## Milestones

### M0 — Proof of Concept (1–2 days)

- [ ] KoreGoOS repo created with `go.mod` importing KoreGo
- [ ] `init`, `mount`, `umount`, `mknod`, `reboot`, `poweroff`, `halt`, `getty` built
- [ ] `cmd/koregoos/main.go` wires blank imports + `korego.Main()`
- [ ] Minimal `/etc/rc`: mount /proc/sys/dev, `echo "booted"`
- [ ] QEMU boots, prints "booted", accepts shell input on console
- [ ] `poweroff` triggers clean QEMU exit

### M1 — Usable System (3–5 days)

- [ ] `login`, `passwd`, `ifconfig`, `route`, `dmesg` built
- [ ] Multi-user support (root + unprivileged user)
- [ ] `/etc/rc` handles networking (loopback + DHCP)
- [ ] JSON-RPC daemon starts at boot
- [ ] All 477 BusyBox tests pass inside the VM
- [ ] CI: QEMU boot smoke test on every push

### M2 — Real Distro (1–2 weeks)

- [ ] `dhclient`, `ping`, `modprobe`, `syslogd`, `crond` built
- [ ] Persistent storage (mount a disk image, survive reboot)
- [ ] `fsck`, `mkfs.ext2`, `fdisk` built
- [ ] Release pipeline: tagged versions → initramfs.cpio.gz artifacts
- [ ] Documentation: quickstart, architecture, security model

---

## Verification

```bash
# Clone and build
git clone https://github.com/ramayac/koregoos
cd koregoos
make

# Boot in QEMU
make qemu

# Inside the VM:
koregoos ls --json /bin
koregoos mount
koregoos ps --json
koregoos ifconfig lo 127.0.0.1 up
koregoos ping 127.0.0.1
koregoos poweroff
```

## Design Observations (2026-05-16)

These observations were collected during the design review and are addressed in
`wiki/prepare_to_goose.md`. They are preserved here for context when adapting this
document for the KoreGoOS repo.

### Shebang quirk

The Linux kernel passes the entire shebang line after `#!` as a **single argument**
with a leading space. `#!/bin/koregoos shell` becomes `exec("/bin/koregoos", " shell", "/etc/rc")`.
The `" shell"` (with space) won't match the dispatch command `"shell"`.

**Mitigations:**
- Trim leading whitespace from argv in the shell command handler
- Or avoid shebang entirely: have `init` invoke `koregoos shell /etc/rc` explicitly (recommended)

### Init complexity is the real risk

The PID 1 `init` handles more than the 7-step boot list suggests:
- **Zombie reaping:** PID 1 inherits orphans. Must `waitpid(-1)` in a loop.
- **Signal forwarding:** SIGINT/SIGTERM must trigger clean shutdown with correct
  unmount ordering. Wrong order → stuck unmounts → kernel panic on reboot.
- **getty respawn:** If getty exits (user logs out, terminal hangup), init must
  respawn it. A single fork works since init IS PID 1.
- **rc failure:** If `/etc/rc` fails (bad syntax, missing mount), init must not
  hang. Log and either drop to emergency shell or continue degraded.

BusyBox `init` is ~1000 lines of C. 200 lines of Go is achievable for a minimal
init, but every edge case needs a test (see "init reaper test" in Test Strategy).

### Shell interpreter dependency

`internal/shell` uses `mvdan.cc/sh/v3` — an external Go module. This is acceptable
(the AGENTS.md carve-out: "unless absolutely necessary, e.g., a complex shell
interpreter later on") but worth noting since it breaks the zero-dependency invariant
for the shell component specifically.

### `--list-commands` and `WellKnownNames` already exist

The public API surface KoreGoOS needs (`korego.WellKnownNames`, `korego.Main()`,
`korego.Run()`, `--list-commands`) is already implemented in `korego.go`. No changes
needed to the dispatch core.

### Daemon works as-is

`korego daemon` is registered and supports `--socket`, `--workers`, `--listen-addr`.
KoreGoOS starts it with `koregoos daemon --socket /run/korego.sock &`. Zero changes needed.

## Related Documents

- [prepare_to_goose.md](prepare_to_goose.md) — Changes needed in KoreGo to support KoreGoOS
- [02_docker_ci.md](02_docker_ci.md) — KoreGo Docker pipeline (same FROM scratch philosophy)
- [05_daemon_core.md](05_daemon_core.md) — JSON-RPC daemon (runs at boot in KoreGoOS)
- [07_agent_features.md](07_agent_features.md) — Shell interpreter (powers /etc/rc)
- [09_release_docs.md](09_release_docs.md) — Release pipeline (model for KoreGoOS releases)
- [korego.go](../korego.go) — Public API that KoreGoOS calls
