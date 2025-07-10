package tg

import (
	"context"
	"fmt"
	"log/slog"

	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type BotController struct {
	me      *bot.Bot
	courts  CourtsService
	storage Storage
}

func NewBotController(
	tgbot *bot.Bot,
	courts CourtsService,
	storage Storage,
) *BotController {
	return &BotController{
		tgbot,
		courts,
		storage,
	}
}

type Actions struct {
	EditMessage *bot.EditMessageTextParams
	SendMessage *bot.SendMessageParams
}

func (b *BotController) HandleMessage(
	ctx context.Context,
	msg *models.Message,
) error {
	var actions *Actions
	var err error
	switch msg.Text {
	case "/commands", "/start":
		var newMsg *UserMessage
		newMsg, err = b.buildCourtCheckerState(
			ctx,
			timeNow(),
			nil,
			nil,
		)

		if err != nil {
			return err
		}

		actions = &Actions{
			SendMessage: &bot.SendMessageParams{
				ChatID:    msg.Chat.ID,
				Text:      newMsg.Markdown,
				ParseMode: models.ParseModeMarkdown,
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: newMsg.Keyboard,
				},
			},
		}
	}

	return b.applyActions(ctx, actions)
}

func (b *BotController) HandleCallbackQuery(
	ctx context.Context,
	cb *models.CallbackQuery,
) error {
	if cb.Message.Message == nil {
		return nil
	}

	state, err := b.getState(ctx, cb.Message.Message)
	if err != nil {
		return err
	}

	parsedCb, err := decodeCallbackData(cb.Data)
	if err != nil {
		return err
	}

	if state == nil {
		state = &tgbotv1.State{
			Ts: int32(timeNow().Unix()),
		}
	}

	ts := parsedCb.GetChangeDatetime().GetDatetime()
	if ts == 0 {
		ts = state.GetTs()
	}

	tm := timeUnix(int64(ts))

	slog.InfoContext(ctx, "callback", "time", tm, "data", parsedCb, "state", state)

	var actions *Actions
	switch {
	case state.GetCheckCourts() != nil:
		actions, err = b.handleCheckCourtsCallbackQuery(
			ctx,
			tm,
			state.GetCheckCourts(),
			parsedCb,
			cb,
		)
	case state.GetCheckCourtsResults() != nil:
		actions, err = b.handleCheckCourtsResultsCallbackQuery(
			ctx,
			tm,
			state,
			parsedCb,
			cb,
		)
	}
	if err != nil {
		return err
	}

	if err := b.applyActions(ctx, actions); err != nil {
		return fmt.Errorf("applyActions: %w", err)
	}

	if _, err := b.me.AnswerCallbackQuery(ctx,
		&bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			ShowAlert:       false,
		}); err != nil {
		return fmt.Errorf("me.AnswerCallbackQuery: %w", err)
	}

	return nil
}
