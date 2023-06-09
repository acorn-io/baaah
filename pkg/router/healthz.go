package router

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"

	"github.com/acorn-io/baaah/pkg/log"
)

var healthz struct {
	healths map[string]bool
	started bool
	lock    *sync.RWMutex
	port    int
}

func init() {
	healthz.lock = &sync.RWMutex{}
	healthz.healths = make(map[string]bool)
}

func setPort(port int) {
	healthz.lock.Lock()
	defer healthz.lock.Unlock()
	if healthz.port > 0 {
		log.Warnf("healthz port cannot be changed")
		return
	}
	healthz.port = port
}

func setHealthy(name string, healthy bool) {
	healthz.lock.Lock()
	defer healthz.lock.Unlock()
	healthz.healths[name] = healthy
}

func getHealthy() bool {
	healthz.lock.RLock()
	defer healthz.lock.RUnlock()
	for _, healthy := range healthz.healths {
		if !healthy {
			return false
		}
	}
	return true
}

// startHealthz starts a healthz server on the healthzPort. If the server is already running, then this is a no-op.
// Similarly, if the healthzPort is <= 0, then this is a no-op.
func startHealthz(ctx context.Context) {
	healthz.lock.Lock()
	defer healthz.lock.Unlock()
	if healthz.started || healthz.port <= 0 {
		return
	}
	healthz.started = true

	// Catch these signals to ensure a graceful shutdown of the server.
	sigCtx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		if getHealthy() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", healthz.port),
		Handler: mux,
	}
	go func() {
		<-sigCtx.Done()
		// Must cancel so that the registered signals are no longer caught.
		cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Warnf("error shutting down healthz server: %v", err)
		}
	}()
	go func() {
		log.Infof("healthz server stopped: %v", srv.ListenAndServe())
	}()
}
