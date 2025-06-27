package klubyorgv1api

import (
	"context"
	"fmt"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	klubyorgv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1"
	"github.com/auvn/go-klubyorg/pkg/gen/proto/klubyorg/v1/klubyorgv1connect"
	"github.com/bufbuild/connect-go"
)

var _ klubyorgv1connect.CourtsServiceHandler = (*Handler)(nil)

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
	in *connect.Request[klubyorgv1.GetCourtsRequest],
) (*connect.Response[klubyorgv1.GetCourtsResponse], error) {
	courts, err := h.svc.GetCourts(
		ctx,
		in.Msg.GetTs().AsTime(),
		in.Msg.GetDuration().AsDuration(),
	)
	if err != nil {
		return nil, fmt.Errorf("svc.GetCourts: %w", err)
	}

	apicourts := make([]*klubyorgv1.GetCourtsResponse_Court, 0, len(courts))
	for _, c := range courts {
		apicourts = append(apicourts,
			&klubyorgv1.GetCourtsResponse_Court{
				ReservationUrl: c.HRef,
				Club: &klubyorgv1.Club{
					ClubName:    c.Club,
					ClubAddress: c.Address,
				},
				CourtPrice: &klubyorgv1.CourtPrice{
					CourtType:  c.Type,
					CourtPrice: c.Price.Text('f', 2),
				},
			},
		)
	}

	return connect.NewResponse(&klubyorgv1.GetCourtsResponse{
		Courts: apicourts,
	}), nil
}
