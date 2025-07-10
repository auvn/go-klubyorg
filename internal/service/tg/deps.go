package tg

import (
	"context"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgstorage"
)

type CourtsService interface {
	GetCourts(
		ctx context.Context,
		ts time.Time,
		duration time.Duration,
	) ([]klubyorg.CourtResult, error)
}

type Storage interface {
	Put(
		ctx context.Context,
		b []byte,
	) (*tgstorage.Receipt, error)
	Get(
		ctx context.Context,
		receipt *tgstorage.Receipt,
	) ([]byte, error)
}
