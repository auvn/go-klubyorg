package tgbundle

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type BotUpdates <-chan *models.Update

func NewBot(
	token string,
	webhookSecretToken string,
) (*bot.Bot, BotUpdates) {
	ch := make(chan *models.Update, 0)
	b, err := bot.New(
		token,
		bot.WithDebug(),
		bot.WithNotAsyncHandlers(),
		bot.WithWebhookSecretToken(webhookSecretToken),
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

type BotUpdatesConfig struct {
	Polling bool
	Webhook *BotUpdatesWebhookConfig
}

type BotUpdatesWebhookConfig struct {
	URL         string
	SecretToken string
}

func (cfg *BotUpdatesWebhookConfig) GetSecretToken() string {
	if cfg == nil {
		return ""
	}

	return cfg.SecretToken
}

func ServeBotUpdates(
	ctx context.Context,
	cfg BotUpdatesConfig,
	b *bot.Bot,
	botcontroller BotController,
	updates BotUpdates,
) error {

	go NewUpdatesHandler(botcontroller, updates)(ctx)

	if cfg.Polling {
		_, err := b.DeleteWebhook(ctx,
			&bot.DeleteWebhookParams{
				DropPendingUpdates: true,
			})
		if err != nil {
			return fmt.Errorf("DeleteWebhook: %w", err)
		}

		b.Start(ctx)

		return nil
	}

	if cfg.Webhook == nil {
		return fmt.Errorf("webhooks config cannot be nil")
	}

	wcfg := cfg.Webhook

	_, err := b.SetWebhook(ctx, &bot.SetWebhookParams{
		URL:         wcfg.URL,
		SecretToken: wcfg.SecretToken,
	})
	if err != nil {
		return fmt.Errorf("set webhook: %w", err)
	}

	b.StartWebhook(ctx)

	return nil
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
