package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/emersion/go-smtp"
	skybackend "github.com/kartverket/skyline/pkg/backend"
	"github.com/kartverket/skyline/pkg/config"
	"github.com/kartverket/skyline/pkg/email"
	"github.com/kartverket/skyline/pkg/smtpd"
	logutils "github.com/kartverket/skyline/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Replacing this with smptd.
type SkylineServer struct {
	ctx        context.Context
	metrics    *http.Server
	mailServer *smtpd.Server
}

var (
	ctx          context.Context
	stop         context.CancelFunc
	gracePeriod  = 30 * time.Second
	globalConfig config.SkylineConfig
)

func init() {
	ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
}

// TODO: This is the part that calls Office365 - probably some re-engineering is needed in this function.
func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	x, y := email.Parse(bytes.NewReader(data))

	return nil
}

// TODO: Test this that it works.
func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	if string(username[:]) == globalConfig.BasicAuthConfig.Username && string(password[:]) == globalConfig.BasicAuthConfig.Password {
		return true, nil
	}
	return false, nil
}

func NewServer(cfg *config.SkylineConfig) *SkylineServer {
	//Basically the handler here will be a fat function which calls the office365 piece and sends the mail.
	globalConfig = *cfg
	addr := cfg.Hostname + ":" + strconv.FormatUint(uint64(cfg.Port), 10)
	appname := "skyline"
	hostname := cfg.Hostname
	srv := smtpd.Server{Addr: addr, Handler: mailHandler, Appname: appname, Hostname: hostname,
		AuthHandler: authHandler, AuthRequired: true}
	if cfg.SSLEnabled == true {
		srv.ConfigureTLS(cfg.SSLCertFile, cfg.SSLPrivateKeyFile)
	}
	skybackend.NewBackend(cfg)

	return &SkylineServer{
		ctx:        ctx,
		metrics:    metricsServer(cfg.MetricsPort),
		mailServer: &srv,
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
		slog.Info("Starting SkylineServer at " + s.mailServer.Addr)
		if err := s.mailServer.ListenAndServe(); err != nil {
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
			err := s.mailServer.Shutdown(shutdownCtx)
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
