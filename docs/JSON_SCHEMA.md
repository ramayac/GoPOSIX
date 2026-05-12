# JSON Output Schemas

All KoreGo utilities support structured machine-readable output via the `--json` flag or when invoked via the JSON-RPC daemon.

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
./korego ls --json /tmp | npx ajv-cli validate -s test/schemas/ls.schema.json

# Run all schema validations against golden fixtures
make validate-schemas
```

## Utility Schemas

Schemas are provided for all 42 utilities that support `--json` output:

| Utility | Data Shape |
|---------|-----------|
| `basename` | `{"result": "string"}` |
| `cat` | `{"lines": ["string"], "lineCount": int}` |
| `chgrp` | `{"changed": [{"path": "string"}]}` |
| `chmod` | `{"changed": [{"path": "string", "mode": "string"}]}` |
| `chown` | `{"changed": [{"path": "string"}]}` |
| `cp` | `{"copied": [{"from": "string", "to": "string"}]}` |
| `cut` | `{"lines": [{"fields": ["string"]}]}` |
| `date` | `{"iso": "string", "unix": int, "utc": "string", "timezone": "string"}` |
| `df` | `[{"filesystem": "string", "size": int, "used": int, "avail": int, "mountpoint": "string"}]` |
| `diff` | `{"files": ["string"], "differ": bool, "hunks": [...]}` |
| `dirname` | `{"result": "string"}` |
| `du` | `[{"path": "string", "size": int, "files": int}]` |
| `echo` | `{"text": "string"}` |
| `env` | `{"vars": {"key": "value", ...}}` |
| `expr` | `{"result": "string", "exitCode": int}` |
| `find` | `[{"path": "string", "type": "string", "size": int, "mtime": "string"}]` |
| `grep` | `[{"file": "string", "line": int, "text": "string", "matches": ["string"]}]` |
| `gzip` | `[{"file": "string", "originalSize": int, "newSize": int, "ratio": number}]` |
| `head` | `{"lines": ["string"], "lineCount": int}` |
| `hostname` | `{"hostname": "string"}` |
| `id` | `{"uid": int, "user": "string", "gid": int, "group": "string", "groups": ["string"]}` |
| `kill` | `{"signaled": [{"pid": int, "signal": "string", "success": bool}]}` |
| `ln` | `{"links": [{"target": "string", "link": "string"}]}` |
| `ls` | `{"path": "string", "files": [...], "total": int}` or `[{...}]` |
| `md5sum` | `[{"file": "string", "hash": "string", "algorithm": "md5"}]` or check mode |
| `mkdir` | `{"created": ["string"]}` |
| `mv` | `{"moved": [{"from": "string", "to": "string"}]}` |
| `printenv` | `{"vars": {"key": "value", ...}}` |
| `printf` | `{"output": "string"}` |
| `ps` | `[{"pid": int, "ppid": int, "user": "string", "cmd": "string", "cpu": "string", "mem": "string"}]` |
| `pwd` | `{"path": "string"}` |
| `readlink` | `{"path": "string", "target": "string"}` |
| `rm` | `{"removed": ["string"], "errors": ["string"]}` |
| `rmdir` | `{"removed": ["string"]}` |
| `sha256sum` | `[{"file": "string", "hash": "string", "algorithm": "sha256"}]` or check mode |
| `sort` | `{"lines": ["string"], "count": int}` |
| `stat` | `{"path": "string", "size": int, "mode": "string", ...}` |
| `tail` | `{"lines": ["string"], "lineCount": int}` |
| `tar` | `[{"name": "string", "size": int, "mode": "string"}]` |
| `test` | `{"result": bool}` |
| `touch` | `{"touched": ["string"]}` |
| `uname` | `{"sysname": "string", "nodename": "string", "release": "string", "version": "string", "machine": "string"}` |
| `uniq` | `[{"line": "string", "count": int}]` |
| `wc` | `{"lines": int, "words": int, "bytes": int, "chars": int}` or multi-file map |
| `whoami` | `{"user": "string", "uid": int}` |
| `xargs` | `[{"command": "string", "exitCode": int}]` |

Utilities without `--json` support: `sleep`, `tee`, `tr`, `yes`, `sed`, `true`, `false`.

## Schema Versioning

The `schemaVersion` field in the envelope allows consumers to detect breaking changes. When a utility's JSON output shape changes incompatibly, the schema version must be bumped (e.g., `"1.0"` → `"2.0"`) and the corresponding schema file updated.

## CI

`make validate-schemas` runs in CI and fails the build if any golden fixture does not validate against its published schema.
