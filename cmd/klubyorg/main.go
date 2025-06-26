package main

import (
	"context"
	"net/http"

	"github.com/auvn/go-app/bootstrap/appx"
	"github.com/auvn/go-app/httpx"
	klubyorgv1api "github.com/auvn/go-klubyorg/internal/api/connect/klubyorg/v1api"
	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/v1connect"
)

func main() {
	app := appx.NewApp()

	app.Go(func(ctx context.Context) error {
		var mux http.ServeMux

		courtsServiceHandler := klubyorgv1api.NewHandler(klubyorg.NewService())

		mux.Handle(v1connect.NewGetCourtsServiceHandler(courtsServiceHandler))

		return httpx.RunServer(ctx, ":8080", &mux)
	})

	app.Run(context.Background())
}
