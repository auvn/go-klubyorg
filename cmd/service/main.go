package main

import (
	"context"
	"net/http"
	"os"

	"github.com/auvn/go-app/bootstrap/appx"
	"github.com/auvn/go-app/httpx"
	"github.com/auvn/go-klubyorg/internal/api/connect/klubyorgv1api"
	"github.com/auvn/go-klubyorg/internal/corsx"
	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	"github.com/auvn/go-klubyorg/internal/service/tg"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgbundle"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgstorage"
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/klubyorgv1connect"
)

type Config struct {
	Telegram struct {
		BotToken      string
		StorageChatID int64
	}
}

func main() {
	var cfg Config

	cfg.Telegram.BotToken = os.Getenv("TG_BOT_TOKEN")
	cfg.Telegram.StorageChatID = -4912380855

	app := appx.NewApp()

	klubySvc := klubyorg.NewService()

	if tgBotToken := cfg.Telegram.BotToken; tgBotToken != "" {
		botapi, botupdates := tgbundle.NewBot(tgBotToken)
		baseStorage := tgstorage.NewStorage(cfg.Telegram.StorageChatID, botapi)
		controller := tg.NewBotController(botapi, klubySvc, baseStorage)

		app.Go(func(ctx context.Context) error {
			go tgbundle.NewUpdatesHandler(
				controller, botupdates,
			)(
				context.WithoutCancel(ctx),
			)

			botapi.Start(ctx)
			return nil
		})
	}

	app.Go(func(ctx context.Context) error {
		var mux http.ServeMux

		courtsServiceHandler := klubyorgv1api.NewHandler(klubySvc)

		mux.Handle(klubyorgv1connect.NewCourtsServiceHandler(courtsServiceHandler))

		h := corsx.ConfigureHandler(&mux)

		return httpx.RunServer(ctx, ":8080", h)
	})

	app.Run(context.Background())
}
