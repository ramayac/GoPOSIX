# Self-Upgrade & Versioning

> **Status:** IMPLEMENTED | **Date:** 2026-05-18
>
> **Goal:** `goposix --version` prints the current build version. `goposix --upgrade`
> fetches the latest GitHub release and atomically replaces the running binary.
> Both work in subcommand mode (`goposix --version`, `goposix --upgrade`).

---

## 1. `--version`

Prints the binary name and version, then exits 0.

```
$ goposix --version
goposix version 1.0.6
```

The version string is injected at build time via `-ldflags`:

```makefile
# Makefile (excerpt)
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X '$(MODULE)/pkg/common.Version=$(VERSION)' \
                      -X 'github.com/ramayac/goposix.Version=$(VERSION)'"
```

Two variables carry the version:

| Variable | Location | Purpose |
|----------|----------|---------|
| `goposix.Version` | `goposix.go` (root package) | Printed by `--version`, compared in `--upgrade` |
| `common.Version` | `pkg/common/output.go` | Embedded in every `--json` output envelope |

Both default to `"0.1.0"` when built without ldflags (e.g., `go build` in development).

---

## 2. `--upgrade`

Self-upgrades the running binary to the latest GitHub release.

```
$ goposix --upgrade
upgrading goposix from 0.1.0 to 1.0.6...
goposix upgraded to 1.0.6
```

### Flow

```
goposix --upgrade
  │
  ├─ 1. os.Executable() + filepath.EvalSymlinks()
  │     Locates the running binary on disk.
  │
  ├─ 2. GET https://api.github.com/repos/ramayac/goposix/releases/latest
  │     Parses JSON: tag_name + assets[].browser_download_url
  │
  ├─ 3. Version comparison (isNewer)
  │     Strips "v" prefix. Compares dotted segments.
  │     Pre-release tags (-rc1, -beta) sort before stable.
  │     Already up-to-date → exits 0, no download.
  │
  ├─ 4. Asset selection
  │     Matches asset name against runtime.GOOS / runtime.GOARCH.
  │     E.g., "_linux_amd64.tar.gz" on Linux/x86-64.
  │
  ├─ 5. Download
  │     GET asset URL → temp file.
  │     If .tar.gz: gzip decompress → tar extract → find "goposix" entry.
  │     If raw binary: direct io.Copy.
  │
  ├─ 6. Atomic replacement
  │     os.Chmod(temp, 0755)
  │     os.Rename(temp, selfPath)   ← atomic on same filesystem
  │
  └─ 7. Exit 0
        Prints "goposix upgraded to X.Y.Z" to stderr.
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Upgrade succeeded, or already at latest version |
| 1 | Upgrade failed (network error, permission denied, no platform asset, etc.) |

### Idempotency

Running `--upgrade` when already at the latest version is a no-op:

```
$ goposix --upgrade
goposix is already at the latest version (1.0.6)
```

### Bootstrap

The `--upgrade` flag is only available in binaries built from a commit that includes
`upgrade.go`. A freshly-built development binary can upgrade itself to the latest
release. But the release binary itself must already contain `--upgrade` code to
self-upgrade to future releases.

After the first release that ships `--upgrade`, the chain is self-sustaining:

```
dev build (has --upgrade) → release v1.X (has --upgrade) → release v2.0 (has --upgrade) → ...
```

---

## 3. Version comparison rules

The `isNewer(a, b)` function in `upgrade.go` handles these cases:

| a | b | isNewer | Reason |
|---|---|-------|--------|
| `1.0.0` | `0.9.0` | `true` | Higher major |
| `0.2.0` | `0.1.0` | `true` | Higher minor |
| `0.1.1` | `0.1.0` | `true` | Higher patch |
| `1.0.0` | `1.0.0` | `false` | Equal |
| `1.0.0` | `1.0` | `true` | More segments |
| `0.1.0` | `0.1.0-rc1` | `true` | Stable > pre-release |
| `abc1234` | `0.1.0` | `false` | Git hash < tagged release |

The GitHub release tag is stripped of its `v` prefix before comparison.

---

## 4. Asset naming

GoReleaser produces assets in this format:

```
GoPOSIX_1.0.6_linux_amd64.tar.gz      ← contains README.md + goposix binary
GoPOSIX_1.0.6_linux_arm64.tar.gz
GoPOSIX_1.0.6_checksums.txt
goposix_1.0.6_linux_amd64.sbom.json   ← SBOM (not a binary, skipped)
```

The upgrade logic matches assets containing `_<os>_<arch>.tar.gz` (case-insensitive
via `strings.Contains`). The tar.gz is expected to contain a file named `goposix`
— the upgrade extracts specifically that entry, ignoring other files like `README.md`.

---

## 5. Files

| File | Role |
|------|------|
| `upgrade.go` | Self-upgrade logic: GitHub API, download, tar/gzip extraction, atomic replacement, version comparison |
| `upgrade_test.go` | Tests for `isNewer`, `parseVersion`, `hasSuffix` |
| `goposix.go` | Wires `--upgrade` into the subcommand-mode dispatcher alongside `--version`, `--help`, `--list-commands` |
| `pkg/common/output.go` | `common.Version` — injected via ldflags, embedded in `--json` envelopes |
| `Makefile` | `VERSION` derivation and `LDFLAGS` injection |
| `.goreleaser.yml` | Release asset naming conventions |

---

## 6. Design decisions

**No external dependencies.** The entire upgrade chain uses only the Go standard
library: `net/http` for GitHub API, `archive/tar` + `compress/gzip` for extraction,
`os.Rename` for atomic replacement. No `go-getter`, `selfupdate`, or other upgrade
libraries.

**Atomic replacement.** `os.Rename` is atomic on Linux when source and destination
are on the same filesystem. The temp file is created in `os.TempDir()` but the
`chmod` + `rename` into the binary's directory is safe because the binary's
directory and temp directory are typically on the same mount. If they aren't,
`os.Rename` returns an error and the upgrade fails cleanly — no half-written binary.

**stderr, not stdout.** All progress messages go to stderr so `--upgrade` can be
used in pipelines without contaminating stdout.

**No daemon interaction.** `--upgrade` works in CLI mode — no daemon required.
It upgrades the binary on disk, which will take effect on the next invocation.

---

## 7. References

- [Phase 22 — Hardening III](22_hardening_iii.md) — Daemon-first pivot (version is embedded in daemon JSON-RPC envelopes)
- [Architecture](architecture.md) — Build pipeline, ldflags injection
- [JSON Schema](json_schema.md) — `common.Version` in the `--json` output envelope
