I'm continuing work on GoPOSIX. Current state:

- Phase 19 (benchmarks): DONE — 10 categories, 288 rows, data-driven
- Phase 20 (hardening II): DONE
- Phase 22 (daemon-first pivot): DONE — docker/Dockerfile is daemon default
- Phase 23 (multi-tenant sandbox): PLANNING — see wiki/23_multi_tenant_sandbox.md

Key benchmark numbers (SCALE=1.0):
- Go SDK daemon: 60µs/call, 10.9× faster than BusyBox fork+exec
- grep on 100MB: 5.1× faster (RE2 engine)
- RPC task loop via SDK: 2.1× faster (5 typed calls/iter, persistent connection)
- BusyBox still wins on: size (11.2×), memory (9.5×), startup (1.8×)

Key invariants:
- Never benchmark daemon through socat — use Go SDK bench_client
- Always run `make bench-quick SCALE=0.1` before `make bench-all`
- Benchmark results live in Docker volume `goposix-bench-data`
- Use `make bench-fetch` to copy results locally
- wiki/phases.md and wiki/index.md track phase status

What should I work on?
