package v1api

import (
	"context"
	"fmt"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	v1 "github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1"
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/v1connect"
	"github.com/bufbuild/connect-go"
)

var _ v1connect.GetCourtsServiceHandler = (*Handler)(nil)

type Service interface {
	GetCourts(
		ctx context.Context,
		ts time.Time,
		duration time.Duration,
	) ([]klubyorg.CourtResult, error)
}

type Handler struct {
	svc Service
}

func NewHandler(
	svc Service,
) *Handler {
	return &Handler{
		svc,
	}
}

func (h *Handler) GetCourts(
	ctx context.Context,
	in *connect.Request[v1.GetCourtsRequest],
) (*connect.Response[v1.GetCourtsResponse], error) {
	courts, err := h.svc.GetCourts(
		ctx,
		in.Msg.GetTs().AsTime(),
		in.Msg.GetDuration().AsDuration(),
	)
	if err != nil {
		return nil, fmt.Errorf("svc.GetCourts: %w", err)
	}

	apicourts := make([]*v1.GetCourtsResponse_Court, 0, len(courts))
	for _, c := range courts {
		apicourts = append(apicourts,
			&v1.GetCourtsResponse_Court{
				ReservationUrl: c.HRef,
				Club: &v1.Club{
					ClubName:    c.Club,
					ClubAddress: c.Address,
				},
				CourtPrice: &v1.CourtPrice{
					CourtType:  c.Type,
					CourtPrice: c.Price,
				},
			},
		)
	}

	return connect.NewResponse(&v1.GetCourtsResponse{
		Courts: apicourts,
	}), nil
}
