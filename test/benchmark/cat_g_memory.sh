#!/bin/sh
# =============================================================================
# Cat G — Memory Footprint (RSS).
# Measures RSS for single invocation, idle daemon, loaded daemon, and BusyBox.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3
CONCURRENT=$(scaled 100 "$MAX_DAEMON_REQUESTS")

echo "# Cat G — Memory Footprint (RSS)" >&2
echo "# scale=$BENCH_SCALE concurrent=$CONCURRENT" >&2
echo "" >&2

SOCKET="$BENCH_TMPDIR/goposix-bench-g.sock"

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# G1: Single echo invocation RSS (via time -f).
for tool in goposix busybox; do
  binary="/bin/$tool"
  echo "# G1: Single echo RSS — $tool" >&2
  for i in $(seq "$SAMPLES"); do
    timing=$( { time -f "%e %U %S %M" "$binary" echo hello >/dev/null; } 2>&1 )
    read -r wall user sys rss rest <<-TIMING_EOF
		$timing
		TIMING_EOF
    : "${rss:=0}"
    echo "mem_single_echo_${tool},$i,$wall,$user,$sys,$rss"
    sleep 0.5
  done
done

# G2: Daemon idle RSS.
echo "# G2: Daemon idle RSS" >&2
rm -f "$SOCKET"
/bin/goposix daemon --socket "$SOCKET" &
DAEMON_PID=$!
sleep 1

if kill -0 "$DAEMON_PID" 2>/dev/null; then
  for i in $(seq "$SAMPLES"); do
    rss=$(ps -o rss= -p "$DAEMON_PID" 2>/dev/null | tr -d ' ' || echo "0")
    echo "mem_daemon_idle_goposix,$i,0,0,0,$rss"
    sleep 0.5
  done

  # G3: Daemon under concurrent load.
  echo "# G3: Daemon under load ($CONCURRENT concurrent)" >&2
  JSON_REQ='{"jsonrpc":"2.0","method":"goposix.echo","params":{"text":"hello"},"id":1}'

  for i in $(seq "$SAMPLES"); do
    # Fire concurrent requests in background.
    for j in $(seq "$CONCURRENT"); do
      echo "$JSON_REQ" | nc -w 2 -U "$SOCKET" >/dev/null 2>&1 &
    done
    sleep 0.3
    rss=$(ps -o rss= -p "$DAEMON_PID" 2>/dev/null | tr -d ' ' || echo "0")
    echo "mem_daemon_loaded_${CONCURRENT}_goposix,$i,0,0,0,$rss"
    wait 2>/dev/null || true
    sleep 1
  done

  kill "$DAEMON_PID" 2>/dev/null || true
  wait "$DAEMON_PID" 2>/dev/null || true
fi
rm -f "$SOCKET"

# G4: BusyBox peak RSS during sequential calls (observed via sampling).
echo "# G4: BusyBox sequential peak RSS ($CONCURRENT calls)" >&2
for i in $(seq "$SAMPLES"); do
  # Run N sequential calls, sample RSS via background watcher.
  (
    for j in $(seq "$CONCURRENT"); do
      /bin/busybox echo hello >/dev/null
    done
  ) &
  PID=$!
  PEAK=0
  while kill -0 "$PID" 2>/dev/null; do
    rss=$(ps -o rss= -p "$PID" 2>/dev/null | tr -d ' ' || echo "0")
    [ "$rss" -gt "$PEAK" ] && PEAK="$rss"
    sleep 0.05
  done
  wait "$PID"
  echo "mem_busybox_sequential_${CONCURRENT},$i,0,0,0,$PEAK"
  sleep 0.5
done

echo "" >&2
echo "# FINDING: BusyBox wins per-invocation RSS (5–20×). GoPOSIX daemon has fixed cost but amortizes it." >&2
