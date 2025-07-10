package tgbundle

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type BotUpdates <-chan *models.Update

func NewBot(
	token string,
) (*bot.Bot, BotUpdates) {
	ch := make(chan *models.Update, 0)
	b, err := bot.New(
		token,
		bot.WithDebug(),
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(func(
			ctx context.Context,
			bot *bot.Bot,
			update *models.Update,
		) {
			select {
			case <-ctx.Done():
				close(ch)
			case ch <- update:
			}
		}),
		bot.WithCheckInitTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}

	return b, ch
}

type BotController interface {
	HandleMessage(ctx context.Context, m *models.Message) error
	HandleCallbackQuery(ctx context.Context, m *models.CallbackQuery) error
}

func NewUpdatesHandler(
	c BotController,
	ch BotUpdates,
) func(ctx context.Context) {
	return func(
		ctx context.Context,
	) {
		for update := range ch {
			slog.Info("update", "data", update)

			var err error
			switch {
			case update.Message != nil:
				err = c.HandleMessage(ctx, update.Message)
			case update.CallbackQuery != nil:
				err = c.HandleCallbackQuery(ctx, update.CallbackQuery)
			}

			if err != nil {
				slog.ErrorContext(ctx, "handler error", "error", err)
			}
		}
	}
}
