package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/deepspace2/plugnpin/pkg/logging"
)

const ENDPOINT = "/metrics"

var log = logging.GetLogger("metricsServer")

func Serve(ctx context.Context, port int, errCh chan<- error) {
	log := log.With("endpoint", ENDPOINT, "port", port)

	mux := http.NewServeMux()
	mux.Handle(ENDPOINT, promhttp.Handler())

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Error("Failed to start metrics server", "error", err)
		errCh <- err
		return
	}

	errCh <- nil

	log.Info("Starting metrics server")

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error("Error serving metrics", "error", err)
		}
	}()

	<-ctx.Done()

	log.Info("Shutting down metrics server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Error shutting down metrics server", "error", err)
	}
}
