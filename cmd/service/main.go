package main

import (
	"context"
	"net/http"

	"github.com/auvn/go-app/bootstrap/appx"
	"github.com/auvn/go-app/httpx"
	"github.com/auvn/go-klubyorg/internal/api/connect/klubyorgv1api"
	"github.com/auvn/go-klubyorg/internal/corsx"
	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/klubyorgv1connect"
)

func main() {
	app := appx.NewApp()

	app.Go(func(ctx context.Context) error {
		var mux http.ServeMux

		courtsServiceHandler := klubyorgv1api.NewHandler(klubyorg.NewService())

		mux.Handle(klubyorgv1connect.NewCourtsServiceHandler(courtsServiceHandler))

		h := corsx.ConfigureHandler(&mux)

		return httpx.RunServer(ctx, ":8080", h)
	})

	app.Run(context.Background())
}
