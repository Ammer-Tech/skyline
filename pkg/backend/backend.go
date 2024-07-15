package backend

import (
	"log/slog"
	"os"

	"github.com/kartverket/skyline/pkg/config"
	skysender "github.com/kartverket/skyline/pkg/sender"
)

// The Backend implements SMTP server methods.
type Backend struct {
	Sender    *skysender.Sender
	BasicAuth *config.BasicAuthConfig
}

func NewBackend(cfg *config.SkylineConfig) *Backend {
	return &Backend{
		Sender:    createSender(cfg),
		BasicAuth: cfg.BasicAuthConfig,
	}
}

func createSender(cfg *config.SkylineConfig) *skysender.Sender {
	var configuredSender skysender.Sender

	switch cfg.SenderType {
	case config.MsGraph:
		sender, err := skysender.NewOffice365Sender(
			cfg.MsGraphConfig.TenantId,
			cfg.MsGraphConfig.ClientId,
			cfg.MsGraphConfig.ClientSecret,
			cfg.MsGraphConfig.SenderUserId,
		)
		if err != nil {
			slog.Error("could not construct sender", "error", err)
			os.Exit(1)
		}
		configuredSender = sender
	case config.Dummy:
		slog.Warn("not implemented yet, exiting cleanly")
		os.Exit(0)
	default:
		slog.Error("unknown sender type", "type", cfg.SenderType)
		os.Exit(1)
	}

	slog.Warn("Was able to build sender!")
	return &configuredSender
}
