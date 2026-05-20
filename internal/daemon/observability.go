package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ramayac/goposix/pkg/common"
)

// Metrics tracks aggregated RPC metrics for Prometheus exposition.
type Metrics struct {
	mu               sync.Mutex
	durationCounts   map[string]int64
	durationSums     map[string]float64 // milliseconds
	rateLimitedTotal int64
}

// NewMetrics returns an initialized Metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		durationCounts: make(map[string]int64),
		durationSums:   make(map[string]float64),
	}
}

// RecordRequest records a completed RPC request duration.
func (m *Metrics) RecordRequest(method string, durationMs float64) {
	m.mu.Lock()
	m.durationCounts[method]++
	m.durationSums[method] += durationMs
	m.mu.Unlock()
}

// RecordRateLimited records a rate-limited request.
func (m *Metrics) RecordRateLimited() {
	atomic.AddInt64(&m.rateLimitedTotal, 1)
}

// ObservabilityServer serves health, readiness, and Prometheus metrics over HTTP.
type ObservabilityServer struct {
	httpServer *http.Server
	addr       string

	// Shared counters (owned by the daemon Server).
	totalRequests *int64
	activeWorkers *int32
	workersMax    int
	connSem       chan struct{}
	uptime        time.Time
	shuttingDown  *int32
	sessionMgr    *SessionManager
	metrics       *Metrics
}

// NewObservabilityServer creates a new HTTP observability server.
func NewObservabilityServer(addr string, totalRequests *int64, activeWorkers *int32,
	workersMax int, connSem chan struct{}, uptime time.Time, shuttingDown *int32, sm *SessionManager, metrics *Metrics,
) *ObservabilityServer {
	o := &ObservabilityServer{
		addr:          addr,
		totalRequests: totalRequests,
		activeWorkers: activeWorkers,
		workersMax:    workersMax,
		connSem:       connSem,
		uptime:        uptime,
		shuttingDown:  shuttingDown,
		sessionMgr:    sm,
		metrics:       metrics,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", o.handleHealthz)
	mux.HandleFunc("/readyz", o.handleReadyz)
	mux.HandleFunc("/metrics", o.handleMetrics)
	mux.HandleFunc("/status", o.handleStatus)
	o.httpServer = &http.Server{Addr: addr, Handler: mux}

	return o
}

// Start begins the HTTP observability server in a goroutine.
func (o *ObservabilityServer) Start() error {
	go func() {
		if err := o.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "observability server error: %v\n", err)
		}
	}()
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (o *ObservabilityServer) Stop() error {
	return o.httpServer.Close()
}

func (o *ObservabilityServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (o *ObservabilityServer) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if atomic.LoadInt32(o.shuttingDown) == 1 {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (o *ObservabilityServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	total := atomic.LoadInt64(o.totalRequests)
	active := atomic.LoadInt32(o.activeWorkers)
	shutdown := atomic.LoadInt32(o.shuttingDown)
	uptimeSec := int64(time.Since(o.uptime).Seconds())
	sessionCount := len(o.sessionMgr.List())
	rateLimited := atomic.LoadInt64(&o.metrics.rateLimitedTotal)

	fmt.Fprintf(w, "# HELP goposix_requests_total Total number of RPC requests processed.\n")
	fmt.Fprintf(w, "# TYPE goposix_requests_total counter\n")
	fmt.Fprintf(w, "goposix_requests_total %d\n", total)

	fmt.Fprintf(w, "# HELP goposix_workers_active Number of currently executing workers.\n")
	fmt.Fprintf(w, "# TYPE goposix_workers_active gauge\n")
	fmt.Fprintf(w, "goposix_workers_active %d\n", active)

	fmt.Fprintf(w, "# HELP goposix_workers_max Configured worker pool size.\n")
	fmt.Fprintf(w, "# TYPE goposix_workers_max gauge\n")
	fmt.Fprintf(w, "goposix_workers_max %d\n", o.workersMax)

	fmt.Fprintf(w, "# HELP goposix_uptime_seconds Daemon uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE goposix_uptime_seconds gauge\n")
	fmt.Fprintf(w, "goposix_uptime_seconds %d\n", uptimeSec)

	fmt.Fprintf(w, "# HELP goposix_sessions_active Number of active sessions.\n")
	fmt.Fprintf(w, "# TYPE goposix_sessions_active gauge\n")
	fmt.Fprintf(w, "goposix_sessions_active %d\n", sessionCount)

	fmt.Fprintf(w, "# HELP goposix_rate_limited_total Total number of rate-limited requests.\n")
	fmt.Fprintf(w, "# TYPE goposix_rate_limited_total counter\n")
	fmt.Fprintf(w, "goposix_rate_limited_total %d\n", rateLimited)

	fmt.Fprintf(w, "# HELP goposix_shutting_down 1 if daemon is draining, 0 otherwise.\n")
	fmt.Fprintf(w, "# TYPE goposix_shutting_down gauge\n")
	fmt.Fprintf(w, "goposix_shutting_down %d\n", shutdown)

	// Go runtime metrics.
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Fprintf(w, "# HELP goposix_goroutines Number of goroutines.\n")
	fmt.Fprintf(w, "# TYPE goposix_goroutines gauge\n")
	fmt.Fprintf(w, "goposix_goroutines %d\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP goposix_gomaxprocs GOMAXPROCS setting.\n")
	fmt.Fprintf(w, "# TYPE goposix_gomaxprocs gauge\n")
	fmt.Fprintf(w, "goposix_gomaxprocs %d\n", runtime.GOMAXPROCS(0))

	fmt.Fprintf(w, "# HELP goposix_num_cpu Number of CPUs available.\n")
	fmt.Fprintf(w, "# TYPE goposix_num_cpu gauge\n")
	fmt.Fprintf(w, "goposix_num_cpu %d\n", runtime.NumCPU())

	fmt.Fprintf(w, "# HELP goposix_heap_alloc_bytes Bytes of allocated heap objects.\n")
	fmt.Fprintf(w, "# TYPE goposix_heap_alloc_bytes gauge\n")
	fmt.Fprintf(w, "goposix_heap_alloc_bytes %d\n", memStats.HeapAlloc)

	fmt.Fprintf(w, "# HELP goposix_heap_sys_bytes Bytes of heap obtained from the OS.\n")
	fmt.Fprintf(w, "# TYPE goposix_heap_sys_bytes gauge\n")
	fmt.Fprintf(w, "goposix_heap_sys_bytes %d\n", memStats.HeapSys)

	fmt.Fprintf(w, "# HELP goposix_stack_inuse_bytes Bytes in stack spans.\n")
	fmt.Fprintf(w, "# TYPE goposix_stack_inuse_bytes gauge\n")
	fmt.Fprintf(w, "goposix_stack_inuse_bytes %d\n", memStats.StackInuse)

	fmt.Fprintf(w, "# HELP goposix_mallocs_total Total number of mallocs.\n")
	fmt.Fprintf(w, "# TYPE goposix_mallocs_total counter\n")
	fmt.Fprintf(w, "goposix_mallocs_total %d\n", memStats.Mallocs)

	fmt.Fprintf(w, "# HELP goposix_frees_total Total number of frees.\n")
	fmt.Fprintf(w, "# TYPE goposix_frees_total counter\n")
	fmt.Fprintf(w, "goposix_frees_total %d\n", memStats.Frees)

	fmt.Fprintf(w, "# HELP goposix_total_alloc_bytes Cumulative bytes allocated for heap objects.\n")
	fmt.Fprintf(w, "# TYPE goposix_total_alloc_bytes counter\n")
	fmt.Fprintf(w, "goposix_total_alloc_bytes %d\n", memStats.TotalAlloc)

	fmt.Fprintf(w, "# HELP goposix_num_gc_cycles Number of completed GC cycles.\n")
	fmt.Fprintf(w, "# TYPE goposix_num_gc_cycles counter\n")
	fmt.Fprintf(w, "goposix_num_gc_cycles %d\n", memStats.NumGC)

	gcPauseNs := memStats.PauseNs[(memStats.NumGC+255)%256]
	fmt.Fprintf(w, "# HELP goposix_gc_pause_ns Most recent GC pause in nanoseconds.\n")
	fmt.Fprintf(w, "# TYPE goposix_gc_pause_ns gauge\n")
	fmt.Fprintf(w, "goposix_gc_pause_ns %d\n", gcPauseNs)

	// Per-method duration aggregates.
	o.metrics.mu.Lock()
	type methodAgg struct {
		method string
		count  int64
		sum    float64
	}
	var methods []methodAgg
	for m, c := range o.metrics.durationCounts {
		methods = append(methods, methodAgg{method: m, count: c, sum: o.metrics.durationSums[m]})
	}
	o.metrics.mu.Unlock()

	if len(methods) > 0 {
		fmt.Fprintf(w, "# HELP goposix_rpc_duration_ms_count Count of RPC calls per method.\n")
		fmt.Fprintf(w, "# TYPE goposix_rpc_duration_ms_count counter\n")
		for _, m := range methods {
			fmt.Fprintf(w, "goposix_rpc_duration_ms_count{method=\"%s\"} %d\n", sanitizeLabel(m.method), m.count)
		}
		fmt.Fprintf(w, "# HELP goposix_rpc_duration_ms_sum Sum of RPC call durations per method in milliseconds.\n")
		fmt.Fprintf(w, "# TYPE goposix_rpc_duration_ms_sum counter\n")
		for _, m := range methods {
			fmt.Fprintf(w, "goposix_rpc_duration_ms_sum{method=\"%s\"} %.2f\n", sanitizeLabel(m.method), m.sum)
		}
	}
}

func sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// StatusSnapshot holds a complete daemon telemetry snapshot.
type StatusSnapshot struct {
	PID        int                  `json:"pid"`
	UptimeSec  int64                `json:"uptime_s"`
	Version    string               `json:"version"`
	Goroutines int                  `json:"goroutines"`
	GOMAXPROCS int                  `json:"gomaxprocs"`
	NumCPU     int                  `json:"num_cpu"`
	Mem        MemSnapshot          `json:"mem"`
	Workers    WorkerSnapshot       `json:"workers"`
	Sessions   SessionStatsSnapshot `json:"sessions"`
	RPC        RPCSnapshot          `json:"rpc"`
	ConnPool   ConnPoolSnapshot     `json:"connection_pool"`
	PerMethod  []MethodSnapshot     `json:"per_method"`
	PerSession []SessionSnapshot    `json:"per_session"`
}

type MemSnapshot struct {
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	StackInuseMB float64 `json:"stack_inuse_mb"`
	GCPauseMs    float64 `json:"gc_pause_ms"`
	NumGC        uint32  `json:"num_gc"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	Mallocs      uint64  `json:"mallocs"`
	Frees        uint64  `json:"frees"`
}

type WorkerSnapshot struct {
	Active int32 `json:"active"`
	Max    int   `json:"max"`
}

type SessionStatsSnapshot struct {
	Active       int   `json:"active"`
	TotalCreated int64 `json:"total_created"`
}

type RPCSnapshot struct {
	TotalCalls  int64 `json:"total_calls"`
	RateLimited int64 `json:"rate_limited"`
}

type ConnPoolSnapshot struct {
	ActiveConns int `json:"active_connections"`
	MaxConns    int `json:"max_connections"`
}

type MethodSnapshot struct {
	Method string  `json:"method"`
	Count  int64   `json:"count"`
	AvgMs  float64 `json:"avg_ms"`
}

type SessionSnapshot struct {
	ID   string `json:"id"`
	AgeS int64  `json:"age_s"`
	CWD  string `json:"cwd"`
}

func bytesToMB(b uint64) float64 {
	return float64(b) / (1024 * 1024)
}

func nsToMs(ns uint64) float64 {
	return float64(ns) / 1_000_000
}

func (o *ObservabilityServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// Per-method aggregates (copy under lock).
	o.metrics.mu.Lock()
	methods := make([]MethodSnapshot, 0, len(o.metrics.durationCounts))
	for method, count := range o.metrics.durationCounts {
		sum := o.metrics.durationSums[method]
		avg := 0.0
		if count > 0 {
			avg = sum / float64(count)
		}
		methods = append(methods, MethodSnapshot{
			Method: method,
			Count:  count,
			AvgMs:  avg,
		})
	}
	o.metrics.mu.Unlock()

	// Per-session details.
	sessions := o.sessionMgr.List()
	now := time.Now()
	perSession := make([]SessionSnapshot, 0, len(sessions))
	for _, s := range sessions {
		perSession = append(perSession, SessionSnapshot{
			ID:   s.ID,
			AgeS: int64(now.Sub(s.LastActive).Seconds()),
			CWD:  s.CWD,
		})
	}

	snapshot := StatusSnapshot{
		PID:        os.Getpid(),
		UptimeSec:  int64(time.Since(o.uptime).Seconds()),
		Version:    common.Version,
		Goroutines: runtime.NumGoroutine(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		NumCPU:     runtime.NumCPU(),
		Mem: MemSnapshot{
			HeapAllocMB:  bytesToMB(mem.HeapAlloc),
			HeapSysMB:    bytesToMB(mem.HeapSys),
			StackInuseMB: bytesToMB(mem.StackInuse),
			GCPauseMs:    nsToMs(mem.PauseNs[(mem.NumGC+255)%256]),
			NumGC:        mem.NumGC,
			TotalAllocMB: bytesToMB(mem.TotalAlloc),
			Mallocs:      mem.Mallocs,
			Frees:        mem.Frees,
		},
		Workers: WorkerSnapshot{
			Active: atomic.LoadInt32(o.activeWorkers),
			Max:    o.workersMax,
		},
		Sessions: SessionStatsSnapshot{
			Active:       len(sessions),
			TotalCreated: o.sessionMgr.TotalCreated(),
		},
		RPC: RPCSnapshot{
			TotalCalls:  atomic.LoadInt64(o.totalRequests),
			RateLimited: atomic.LoadInt64(&o.metrics.rateLimitedTotal),
		},
		ConnPool: ConnPoolSnapshot{
			ActiveConns: len(o.connSem),
			MaxConns:    cap(o.connSem),
		},
		PerMethod:  methods,
		PerSession: perSession,
	}

	json.NewEncoder(w).Encode(snapshot)
}
