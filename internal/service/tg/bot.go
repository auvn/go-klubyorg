package tg

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	me     *bot.Bot
	courts CourtsService
}

func MustNewBot(
	token string,
	courts CourtsService,
) *Bot {
	b, err := NewBot(
		token,
		courts,
	)
	if err != nil {
		panic(err)
	}

	return b
}

func NewBot(
	token string,
	courts CourtsService,
) (*Bot, error) {
	wrapper := Bot{
		nil,
		courts,
	}
	b, err := bot.New(
		token,
		bot.WithDebug(),
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(wrapper.handler),
		bot.WithCheckInitTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("bot.New: %w", err)
	}

	wrapper.me = b

	return &Bot{
		b,
		courts,
	}, nil
}

func (b *Bot) Serve(ctx context.Context) error {
	b.me.Start(ctx)
	return nil
}

func (b *Bot) handler(
	ctx context.Context,
	_ *bot.Bot,
	upd *models.Update,
) {
	slog.Info("update", "data", upd)

	var err error
	switch {
	case upd.Message != nil:
		err = b.handleMessage(ctx, upd.Message)
	case upd.CallbackQuery != nil:
		err = b.handleCallbackQuery(ctx, upd.CallbackQuery)
	}

	if err != nil {
		slog.ErrorContext(ctx, "handler error", "error", err)
	}
}

func (b *Bot) handleMessage(
	ctx context.Context,
	msg *models.Message,
) error {
	switch msg.Text {
	case "/commands", "/start":
		courtChecker, err := b.buildCourtChecker(
			msg.Chat.ID,
			timeNow(),
			nil,
			nil,
		)
		if err != nil {
			return err
		}
		_, err = b.me.SendMessage(ctx, courtChecker)
		return err
	}
	return nil
}

func (b *Bot) handleCallbackQuery(
	ctx context.Context,
	cb *models.CallbackQuery,
) error {
	if cb.Message.Message == nil {
		return nil
	}

	state, err := b.getState(cb.Message.Message)
	if err != nil {
		return err
	}

	parsedCb, err := decodeCallbackData(cb.Data)
	if err != nil {
		return err
	}

	switch {
	case parsedCb.GetReset_() != nil:
		state = parsedCb.GetReset_()
	}

	slog.InfoContext(ctx, "callback", "data", parsedCb, "state", state)

	switch {
	case state.GetCheckCourts() != nil:
		return b.handleCheckCourtsCallbackQuery(ctx, state, parsedCb, cb)
	}

	return nil
}
