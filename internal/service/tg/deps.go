package tg

import (
	"context"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
)

type CourtsService interface {
	GetCourts(
		ctx context.Context,
		ts time.Time,
		duration time.Duration,
	) ([]klubyorg.CourtResult, error)
}
