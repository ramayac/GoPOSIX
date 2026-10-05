---
status: current
description: "The --json output envelope and per-utility schemas."
---

# JSON Output Schemas

All GoPOSIX utilities support structured machine-readable output via the `--json` flag or when invoked via the JSON-RPC daemon.

## Standard Envelope

Every successful utility execution outputs a standard JSON envelope (schema version `1.0`):

```json
{
  "command": "ls",
  "version": "0.1.0",
  "schemaVersion": "1.0",
  "exitCode": 0,
  "data": { ... utility specific data ... },
  "error": null
}
```

On error, `data` is `null` and `error` contains the error details:

```json
{
  "command": "cat",
  "version": "0.1.0",
  "schemaVersion": "1.0",
  "exitCode": 1,
  "data": null,
  "error": {
    "code": "ENOENT",
    "message": "no such file or directory: /nope"
  }
}
```

## Schema Files

Published JSON Schema (draft-07) files live in `test/schemas/`. Each utility has its own schema — e.g., `test/schemas/ls.schema.json`.

The common envelope schema is at `test/schemas/common.schema.json`.

## Validation

```bash
# Validate a single utility's output against its schema
./goposix ls --json /tmp | npx ajv-cli validate -s test/schemas/ls.schema.json

# Run all schema validations against golden fixtures
make validate-schemas
```

## Utility Schemas

Schemas are provided for all 114 utilities that support `--json` output (including the `test`, `true`, and `false` alias names):

| Utility | Data Shape |
|---------|-----------|
| `ar` | `{"archive": "string", "members": [{"name": "string", "size": int, "mod_time": int, "mode": int, "uid": int, "gid": int}]}` |
| `awk` | `{"lines": ["string"], "lineCount": int, "status": int}` |
| `basename` | `{"result": "string"}` |
| `bc` | `{"lines": ["string"]}` |
| `bunzip2` | `{"files": [{"source": "string", "destination": "string", "bytesResult": int, "error": "string"}]}` |
| `bzcat` | `{"files": [{"source": "string", "bytesResult": int, "error": "string"}]}` |
| `cal` | `{"year": int, "month": int, "julian": bool, "monday_start": bool, "calendar": "string"}` |
| `cat` | `{"lines": ["string"], "lineCount": int}` |
| `chgrp` | `{"changed": [{"path": "string"}]}` |
| `chmod` | `{"changed": [{"path": "string", "mode": "string"}]}` |
| `chown` | `{"changed": [{"path": "string"}]}` |
| `cksum` | `{"files": [{"name": "string", "checksum": int, "bytes": int}]}` |
| `cmp` | `{"equal": bool, "byte_pos": int, "line_num": int, "val1": int, "val2": int}` |
| `comm` | `{"only_file1": ["string"], "only_file2": ["string"], "both": ["string"]}` |
| `cp` | `{"copied": [{"from": "string", "to": "string"}]}` |
| `cpio` | `{"members": [{"name": "string", "size": int, "mode": int, "mod_time": int}]}` |
| `cryptpw` | `{"password": "string", "method": "string", "salt": "string", "hash": "string"}` |
| `cut` | `{"lines": [{"fields": ["string"]}]}` |
| `dc` | `{"output": ["string"]}` or null |
| `date` | `{"iso": "string", "unix": int, "utc": "string", "timezone": "string"}` |
| `df` | `[{"filesystem": "string", "size": int, "used": int, "avail": int, "mountpoint": "string"}]` |
| `diff` | `{"files": ["string"], "differ": bool, "hunks": [...]}` |
| `dirname` | `{"result": "string"}` |
| `du` | `[{"path": "string", "size": int, "files": int}]` |
| `echo` | `{"text": "string"}` |
| `env` | `{"vars": {"key": "value", ...}}` |
| `expand` | `{"lines": ["string"]}` |
| `expr` | `{"result": "string", "exitCode": int}` |
| `factor` | `{"results": [{"input": "string", "factors": [int], "error": "string"}]}` |
| `false` | `{"exitCode": int, "value": bool}` |
| `find` | `[{"path": "string", "type": "string", "size": int, "mtime": "string"}]` |
| `fold` | `{"lines": ["string"]}` |
| `grep` | `[{"file": "string", "line": int, "text": "string", "matches": ["string"]}]` |
| `gzip` | `[{"file": "string", "originalSize": int, "newSize": int, "ratio": number}]` |
| `head` | `{"lines": ["string"], "lineCount": int}` |
| `hexdump` | `{"lines": ["string"]}` |
| `hostid` | `{"hostid": "string"}` |
| `hostname` | `{"hostname": "string"}` |
| `id` | `{"uid": int, "user": "string", "gid": int, "group": "string", "groups": ["string"]}` |
| `join` | `{"records": [{"key": "value"}]}` |
| `kill` | `{"signaled": [{"pid": int, "signal": "string", "success": bool}]}`; `-l` mode: `{"signals": ["KILL", ...]}` |
| `link` | `{"source": "string", "target": "string"}` |
| `ln` | `{"links": [{"target": "string", "link": "string"}]}` |
| `logger` | `{"priority": "string", "tag": "string", "message": "string"}` |
| `logname` | `{"logname": "string"}` |
| `ls` | `{"path": "string", "files": [...], "total": int}` or `[{...}]` |
| `makedevs` | `{"table": "string", "rootdir": "string", "created": [{"name": "string", "type": "string", "mode": "string", "uid": int, "gid": int, "major": int, "minor": int, "status": "string", "error": "string"}], "failedCount": int}` |
| `mdev` | `{"devices": [{"name": "string", "type": "string", "major": int, "minor": int, "path": "string"}]}` |
| `md5sum` | `[{"file": "string", "hash": "string", "algorithm": "md5"}]` or check mode |
| `mkfs.minix` | `{"inodes": int, "zones": int, "first_data_zone": int, "imap_blocks": int, "zmap_blocks": int}` |
| `mkdir` | `{"created": ["string"]}` |
| `mkfifo` | `{"path": "string", "mode": "string"}` |
| `mount` | `{"mounts": [{"device": "string", "mountpoint": "string", "fstype": "string", "options": "string"}]}` |
| `mv` | `{"moved": [{"from": "string", "to": "string"}]}` |
| `nice` | `{"adjustment": int, "command": ["string"], "exit_code": int}` |
| `nl` | `{"lines": [{"number": int, "text": "string"}]}` |
| `nohup` | `{"command": ["string"], "output_file": "string", "exit_code": int}` |
| `od` | `{"records": ["string"]}` |
| `paste` | `{"records": [["string"]]}` |
| `patch` | `{"file": "string", "applied": int, "rejected": int, "is_new": bool, "message": "string"}` |
| `pidof` | `{"pids": [int]}` |
| `printenv` | `{"vars": {"key": "value", ...}}` |
| `printf` | `{"output": "string"}` |
| `ps` | `[{"pid": int, "ppid": int, "user": "string", "cmd": "string", "cpu": "string", "mem": "string"}]` |
| `pwd` | `{"path": "string"}` |
| `readlink` | `{"path": "string", "target": "string"}` |
| `realpath` | `{"resolved": {"path": "string"}}` |
| `rev` | `{"lines": ["string"], "line_count": int}` |
| `rm` | `{"removed": ["string"], "errors": ["string"]}` |
| `rmdir` | `{"removed": ["string"]}` |
| `rx` | `{"bytesWritten": int, "fileName": "string"}` or null |
| `sed` | `{"lines": ["string"], "lineCount": int, "changed": bool, "scripts": ["string"]}` |
| `seq` | `{"sequence": ["string"]}` |
| `sha1sum` | `[{"file": "string", "hash": "string", "algorithm": "sha1"}]` or check mode |
| `sha256sum` | `[{"file": "string", "hash": "string", "algorithm": "sha256"}]` or check mode |
| `sha3sum` | `[{"file": "string", "hash": "string", "algorithm": "string"}]` or check mode |
| `sha512sum` | `[{"file": "string", "hash": "string", "algorithm": "sha512"}]` or check mode |
| `shell` | `{"exitCode": int, "stdout": "string", "stderr": "string"}` |
| `sleep` | `{"duration": number, "requested": number, "interrupted": bool}` |
| `sort` | `{"lines": ["string"], "count": int}` |
| `split` | `{"files": ["string"], "chunks": int}` |
| `start-stop-daemon` | `{"action": "string", "matchedPids": [int], "newPid": int, "commandLine": ["string"], "status": "string"}` |
| `stat` | `{"path": "string", "size": int, "mode": "string", ...}` |
| `strings` | `{"strings": [{"offset": int, "value": "string"}]}` |
| `sum` | `{"files": [{"file": "string", "checksum": int, "blocks": int}]}` |
| `tail` | `{"lines": ["string"], "lineCount": int}` |
| `taskset` | `{"pid": int, "currentMask": "string", "newMask": "string", "command": "string"}` |
| `tar` | `[{"name": "string", "size": int, "mode": "string"}]` |
| `tee` | `{"bytesWritten": int, "files": ["string"]}` |
| `test` | `{"result": bool}` |
| `touch` | `{"touched": ["string"]}` |
| `tr` | `{"lines": ["string"], "lineCount": int, "bytesIn": int, "bytesOut": int}` |
| `tree` | `{"trees": [{"name": "string", "type": "string", "target": "string", "contents": [...]}], "report": {"directories": int, "files": int}}` |
| `true` | `{"exitCode": int, "value": bool}` |
| `tsort` | `{"nodes": ["string"]}` or null |
| `tty` | `{"is_tty": bool, "path": "string"}` |
| `uname` | `{"sysname": "string", "nodename": "string", "release": "string", "version": "string", "machine": "string"}` |
| `uncompress` | `{"files": [{"source": "string", "destination": "string", "bytesResult": int, "error": "string"}]}` |
| `unexpand` | `{"lines": ["string"]}` |
| `unlzma` | `{"files": [{"source": "string", "destination": "string", "bytesResult": int, "error": "string"}]}` |
| `uniq` | `[{"line": "string", "count": int}]` |
| `unlink` | `{"removed": "string"}` |
| `unzip` | `{"archive": "string", "files": [{"name": "string", "size": int, "compressedSize": int, "isDir": bool}]}` |
| `uptime` | `{"current_time": "string", "uptime": number, "users": int, "load_1m": number, "load_5m": number, "load_15m": number}` |
| `uudecode` | `{"source": "string", "destination": "string", "bytesDecoded": int}` |
| `uuencode` | `{"source": "string", "remoteName": "string", "encodedData": "string", "format": "string"}` |
| `wc` | `{"lines": int, "words": int, "bytes": int, "chars": int}` or multi-file map |
| `wget` | `{"url": "string", "output_file": "string", "bytes_downloaded": int, "status_code": int}` |
| `which` | `{"matches": {"name": ["path"]}}` |
| `who` | `{"users": [{"name": "string", "terminal": "string", "time": "string", "host": "string"}], "count": int}` |
| `whoami` | `{"user": "string", "uid": int}` |
| `xargs` | `[{"command": "string", "exitCode": int}]` |
| `xxd` | `{"lines": ["string"]}` |
| `yes` | `{"string": "string", "count": int, "truncated": bool}` |

Only `dd` does not yet support `--json` output. The `daemon` control command is also exempt: `--json` and JSON-RPC do not apply to it (it manages the daemon process itself).

## Schema Versioning

The `schemaVersion` field in the envelope allows consumers to detect breaking changes. When a utility's JSON output shape changes incompatibly, the schema version must be bumped (e.g., `"1.0"` → `"2.0"`) and the corresponding schema file updated.

## CI

`make validate-schemas` runs in CI and fails the build if any golden fixture does not validate against its published schema.

---

## See Also

- [index.md](index.md) | Wiki index.
- [usage.md](usage.md) | CLI and daemon usage.
- [rpc_quickstart.md](rpc_quickstart.md) | JSON-RPC protocol reference.
- [schema.md](schema.md) | Wiki structure contract.
