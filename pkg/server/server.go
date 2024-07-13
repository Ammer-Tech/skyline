package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/emersion/go-smtp"
	skybackend "github.com/kartverket/skyline/pkg/backend"
	"github.com/kartverket/skyline/pkg/config"
	"github.com/kartverket/skyline/pkg/smtpd"
	logutils "github.com/kartverket/skyline/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Replacing this with smptd.
type SkylineServer struct {
	ctx     context.Context
	smtpd   *smtpd.Server
	metrics *http.Server
}

var (
	ctx         context.Context
	stop        context.CancelFunc
	gracePeriod = 30 * time.Second
)

func init() {
	ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
}

func NewServer(cfg *config.SkylineConfig) *SkylineServer {
	//Basically the handler here will be a fat function which calls the office365 piece and sends the mail.
	smtpd.ListenAndServeTLS(cfg.Port, cfg.SSLCertFile, cfg.SSLPrivateKeyFile, nilfunc, "skyline", cfg.Hostname)
	skybackend.NewBackend(cfg)

	return &SkylineServer{
		ctx:     ctx,
		metrics: metricsServer(cfg.MetricsPort),
	}
}

func metricsServer(metricsPort uint) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", metricsPort),
		Handler: mux,
	}
}

func (s *SkylineServer) Serve() {
	defer stop()

	go func() {
		slog.Info("Starting SkylineServer at " + s.smtp.Addr)
		if err := s.smtp.ListenAndServe(); err != nil {
			slog.Error("could not start SMTP server", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("Serving metrics at " + s.metrics.Addr)
		if err := s.metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("could not start metrics server", "error", err)
			os.Exit(1)
		}
	}()

	select {
	case <-s.ctx.Done():
		shutdownCtx, _ := context.WithDeadline(context.Background(), time.Now().Add(gracePeriod))
		var wg sync.WaitGroup

		slog.Info("received interrupt, shutting down with a grace period", "duration", gracePeriod)
		wg.Add(2)

		go func() {
			defer wg.Done()
			slog.Info("shutting down SMTP server")
			err := s.smtp.Shutdown(shutdownCtx)
			if err != nil {
				slog.Warn("could not shut down SMTP server", "error", err)
			}
		}()

		go func() {
			defer wg.Done()
			slog.Info("shutting down metrics server")
			err := s.metrics.Shutdown(shutdownCtx)
			if err != nil {
				slog.Warn("could not shut down metrics server", "error", err)
			}
		}()

		wg.Wait()
		slog.Info("shutdown complete")
	}
}

func ioWriterAdapter(ctx context.Context) io.Writer {
	return logutils.NewSlogWriter(
		ctx,
		slog.LevelDebug,
		map[string]string{"component": "smtp", "raw": "true"},
		func(line string) string {
			return strings.Replace(line, "\r", "", 1)
		},
	)
}

func logAdapter(ctx context.Context) smtp.Logger {
	return logutils.NewLogAdapter(
		ctx,
		slog.LevelError,
		map[string]string{"component": "smtp"},
	)
}
