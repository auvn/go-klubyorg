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
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/klubyorgv1connect"
)

func main() {
	app := appx.NewApp()

	klubySvc := klubyorg.NewService()

	if tgBotToken := os.Getenv("TG_BOT_TOKEN"); tgBotToken != "" {
		tgBot := tg.MustNewBot(tgBotToken, klubySvc)

		app.Go(func(ctx context.Context) error {
			return tgBot.Serve(ctx)
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
