package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks aggregated RPC metrics for Prometheus exposition.
type Metrics struct {
	mu              sync.Mutex
	durationCounts  map[string]int64
	durationSums    map[string]float64 // milliseconds
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
	uptime        time.Time
	shuttingDown  *int32
	sessionMgr    *SessionManager
	metrics       *Metrics
}

// NewObservabilityServer creates a new HTTP observability server.
func NewObservabilityServer(addr string, totalRequests *int64, activeWorkers *int32,
	workersMax int, uptime time.Time, shuttingDown *int32, sm *SessionManager, metrics *Metrics,
) *ObservabilityServer {
	o := &ObservabilityServer{
		addr:          addr,
		totalRequests: totalRequests,
		activeWorkers: activeWorkers,
		workersMax:    workersMax,
		uptime:        uptime,
		shuttingDown:  shuttingDown,
		sessionMgr:    sm,
		metrics:       metrics,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", o.handleHealthz)
	mux.HandleFunc("/readyz", o.handleReadyz)
	mux.HandleFunc("/metrics", o.handleMetrics)
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

	fmt.Fprintf(w, "# HELP korego_requests_total Total number of RPC requests processed.\n")
	fmt.Fprintf(w, "# TYPE korego_requests_total counter\n")
	fmt.Fprintf(w, "korego_requests_total %d\n", total)

	fmt.Fprintf(w, "# HELP korego_workers_active Number of currently executing workers.\n")
	fmt.Fprintf(w, "# TYPE korego_workers_active gauge\n")
	fmt.Fprintf(w, "korego_workers_active %d\n", active)

	fmt.Fprintf(w, "# HELP korego_workers_max Configured worker pool size.\n")
	fmt.Fprintf(w, "# TYPE korego_workers_max gauge\n")
	fmt.Fprintf(w, "korego_workers_max %d\n", o.workersMax)

	fmt.Fprintf(w, "# HELP korego_uptime_seconds Daemon uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE korego_uptime_seconds gauge\n")
	fmt.Fprintf(w, "korego_uptime_seconds %d\n", uptimeSec)

	fmt.Fprintf(w, "# HELP korego_sessions_active Number of active sessions.\n")
	fmt.Fprintf(w, "# TYPE korego_sessions_active gauge\n")
	fmt.Fprintf(w, "korego_sessions_active %d\n", sessionCount)

	fmt.Fprintf(w, "# HELP korego_rate_limited_total Total number of rate-limited requests.\n")
	fmt.Fprintf(w, "# TYPE korego_rate_limited_total counter\n")
	fmt.Fprintf(w, "korego_rate_limited_total %d\n", rateLimited)

	fmt.Fprintf(w, "# HELP korego_shutting_down 1 if daemon is draining, 0 otherwise.\n")
	fmt.Fprintf(w, "# TYPE korego_shutting_down gauge\n")
	fmt.Fprintf(w, "korego_shutting_down %d\n", shutdown)

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
		fmt.Fprintf(w, "# HELP korego_rpc_duration_ms_count Count of RPC calls per method.\n")
		fmt.Fprintf(w, "# TYPE korego_rpc_duration_ms_count counter\n")
		for _, m := range methods {
			fmt.Fprintf(w, "korego_rpc_duration_ms_count{method=\"%s\"} %d\n", m.method, m.count)
		}
		fmt.Fprintf(w, "# HELP korego_rpc_duration_ms_sum Sum of RPC call durations per method in milliseconds.\n")
		fmt.Fprintf(w, "# TYPE korego_rpc_duration_ms_sum counter\n")
		for _, m := range methods {
			fmt.Fprintf(w, "korego_rpc_duration_ms_sum{method=\"%s\"} %.2f\n", m.method, m.sum)
		}
	}
}
