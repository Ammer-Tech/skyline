package backend

import (
	"bytes"
	"log"
	"log/slog"
	"net"
	"net/mail"
	"os"

	"github.com/kartverket/skyline/pkg/config"
	"github.com/kartverket/skyline/pkg/email"
	skysender "github.com/kartverket/skyline/pkg/sender"
	"github.com/kartverket/skyline/pkg/smtpd"
)

// The Backend implements SMTP server methods.
type Backend struct {
	Sender    skysender.Sender
	BasicAuth *config.BasicAuthConfig
	Server    *smtpd.Server
}

func forwardToMicrosoft(origin net.Addr, from string, to []string, data []byte) (email.SkylineEmail, error) {
	msg, _ := mail.ReadMessage(bytes.NewReader(data))
	subject := msg.Header.Get("Subject")
	log.Printf("Received mail from %s for %s with subject %s", from, to[0], subject)
	//PUSH TO Send in Office365.go
	return email.SkylineEmail{}, nil
}

func NewBackend(cfg *config.SkylineConfig, server *smtpd.Server) *Backend {
	return &Backend{
		Sender:    createSender(cfg),
		BasicAuth: cfg.BasicAuthConfig,
	}
}

func createSender(cfg *config.SkylineConfig) skysender.Sender {
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

	return configuredSender
}
