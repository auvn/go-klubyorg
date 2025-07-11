package main

import (
	"context"
	"net/http"
	"os"

	"github.com/auvn/go-app/bootstrap/appx"
	"github.com/auvn/go-app/httpx"
	"github.com/auvn/go-klubyorg/internal/corsx"
	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	"github.com/auvn/go-klubyorg/internal/service/tg"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgbundle"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgstorage"
)

type Config struct {
	Telegram struct {
		StorageChatID int64
		BotToken      string
		Updates       tgbundle.BotUpdatesConfig
	}
}

func main() {
	var cfg Config

	cfg.Telegram.BotToken = os.Getenv("TG_BOT_TOKEN")
	cfg.Telegram.StorageChatID = -4912380855
	cfg.Telegram.Updates.Polling = false
	cfg.Telegram.Updates.Webhook = &tgbundle.BotUpdatesWebhookConfig{
		URL:         "https://go-klubyorg.fly.dev/webhooks/tg",
		SecretToken: "superpupersecrettoken",
	}

	app := appx.NewApp()

	var webhooksMux http.ServeMux

	klubySvc := klubyorg.NewService()

	if tgBotToken := cfg.Telegram.BotToken; tgBotToken != "" {
		botapi, botupdates := tgbundle.NewBot(
			tgBotToken,
			cfg.Telegram.Updates.Webhook.GetSecretToken(),
		)
		baseStorage := tgstorage.NewStorage(cfg.Telegram.StorageChatID, botapi)
		controller := tg.NewBotController(botapi, klubySvc, baseStorage)

		webhooksMux.Handle("/tg", botapi.WebhookHandler())

		app.Go(func(ctx context.Context) error {
			return tgbundle.ServeBotUpdates(
				ctx,
				cfg.Telegram.Updates,
				botapi,
				controller,
				botupdates,
			)
		})
	}

	app.Go(func(ctx context.Context) error {
		var mux http.ServeMux

		mux.Handle("/webhooks", http.StripPrefix("/webhooks", &webhooksMux))

		h := corsx.ConfigureHandler(&mux)

		return httpx.RunServer(ctx, ":8080", h)
	})

	app.Run(context.Background())
}
