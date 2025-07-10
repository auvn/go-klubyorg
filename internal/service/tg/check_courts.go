package tg

import (
	"context"
	"fmt"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/tg/dataenc"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgstorage"
	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	_30mins = 30

	_buttonCheckCourts = "Check Courts"
	_textCheckCourts   = "Select a date 📅 and duration 🕒 then hit the Check Courts button"
)

func (b *BotController) handleCheckCourtsResultsCallbackQuery(
	ctx context.Context,
	tm time.Time,
	state *tgbotv1.State,
	cbData *tgbotv1.Callbacks_Data,
	cb *models.CallbackQuery,
) (*Actions, error) {
	switch {
	case cbData.GetUpdatePager() != nil:
		storedClubs, err := b.storage.Get(ctx,
			&tgstorage.Receipt{
				MessageID: int(state.GetCheckCourtsResults().GetFileReceipt().GetMessageId()),
			},
		)

		if err != nil {
			return nil, err
		}

		var clubs tgbotv1.State_AvailableCourts
		if err := dataenc.Decode(storedClubs, &clubs); err != nil {
			return nil, fmt.Errorf("decode stored clubs: %w", err)
		}

		date := timeUnix(int64(state.GetCheckCourtsResults().GetParams().GetDatetime().GetDatetime()))
		newPager := cbData.GetUpdatePager()
		userMessage := buildCheckCourtsResultsMessage(
			date,
			time.Hour,
			state,
			newPager,
			clubs.Clubs,
		)
		return &Actions{
			EditMessage: &bot.EditMessageTextParams{
				ChatID:    cb.Message.Message.Chat.ID,
				MessageID: cb.Message.Message.ID,
				Text:      userMessage.Markdown,
				ParseMode: models.ParseModeMarkdown,
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: userMessage.Keyboard,
				},
			},
		}, nil
	}

	return nil, nil
}

func (b *BotController) handleCheckCourtsCallbackQuery(
	ctx context.Context,
	tm time.Time,
	state *tgbotv1.State_CheckCourts,
	cbData *tgbotv1.Callbacks_Data,
	cb *models.CallbackQuery,
) (*Actions, error) {
	if cbData.GetFinalize() != nil {
		hourHalfs := state.GetDuration().GetHourHalfs()
		date := timeUnix(int64(state.GetDatetime().GetDatetime()))
		duration := time.Duration(hourHalfs*_30mins) * time.Minute
		result, err := b.courts.GetCourts(
			ctx,
			date,
			duration,
		)
		if err != nil {
			return nil, fmt.Errorf("courts.GetCourts: %w", err)
		}

		clubs := collectAvailableCourts(result)

		fileReceipt, err := b.storage.Put(ctx, dataenc.Encode(clubs))
		if err != nil {
			return nil, fmt.Errorf("storage.Put: %w", err)
		}

		newState := tgbotv1.State{
			Ts: int32(tm.Unix()),
			V: &tgbotv1.State_CheckCourtsResults_{
				CheckCourtsResults: &tgbotv1.State_CheckCourtsResults{
					Params: state,
					Pager: &tgbotv1.Callbacks_Pager{
						Limit: 5,
					},
					FileReceipt: &tgbotv1.StorageMetadata_FileReceipt{
						MessageId: int64(fileReceipt.MessageID),
					},
				},
			},
		}

		userMessage := buildCheckCourtsResultsMessage(
			date,
			duration,
			&newState,
			newState.GetCheckCourtsResults().GetPager(),
			clubs.Clubs,
		)

		return &Actions{
			EditMessage: &bot.EditMessageTextParams{
				ChatID:    cb.Message.Message.Chat.ID,
				MessageID: cb.Message.Message.ID,
				Text:      userMessage.Markdown,
				ParseMode: models.ParseModeMarkdown,
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: userMessage.Keyboard,
				},
			},
		}, nil
	}

	courtChecker, err := b.buildCourtCheckerState(
		ctx,
		tm,
		state,
		cbData,
	)
	if err != nil {
		return nil, err
	}

	return &Actions{
		EditMessage: &bot.EditMessageTextParams{
			ChatID:    cb.Message.Message.Chat.ID,
			MessageID: cb.Message.Message.ID,
			Text:      courtChecker.Markdown,
			ParseMode: models.ParseModeMarkdown,
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: courtChecker.Keyboard,
			},
		},
	}, nil
}

type CourtChecker struct {
	Message            string
	Keyboard           [][]models.InlineKeyboardButton
	State              *tgbotv1.State
	FinalizeButtonText string
}

type UserMessage struct {
	Markdown string
	Keyboard [][]models.InlineKeyboardButton
}

func (b *BotController) buildCourtCheckerState(
	ctx context.Context,
	tm time.Time,
	state *tgbotv1.State_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) (*UserMessage, error) {
	checker, err := b.buildCourtChecker(tm, state, cb)
	if err != nil {
		return nil, fmt.Errorf("build court checker: %w", err)
	}

	if checker.FinalizeButtonText != "" {
		checker.Keyboard = append(checker.Keyboard, []models.InlineKeyboardButton{
			{
				Text:         checker.FinalizeButtonText,
				CallbackData: encodeCallbackData(finalizeCallback()),
			},
		})
	}

	text, _ := messageTextWithState(checker.Message, checker.State)
	return &UserMessage{
		Markdown: text,
		Keyboard: checker.Keyboard,
	}, nil
}

func (b *BotController) buildCourtChecker(
	tm time.Time,
	state *tgbotv1.State_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) (*CourtChecker, error) {

	tm = tm.Round(30 * time.Minute)

	kb := [][]models.InlineKeyboardButton{}

	calendarPicker, selectedTime := buildDayPicker(tm)
	kb = append(kb, calendarPicker...)

	durationPicker, selectedDuration := buildDurationPicker(state, cb)
	kb = append(kb, durationPicker)

	finalState := checkCourtsState(
		tm,
		selectedTime,
		selectedDuration,
		0,
	)

	return &CourtChecker{
		Message:            _textCheckCourts,
		Keyboard:           kb,
		State:              finalState,
		FinalizeButtonText: _buttonCheckCourts,
	}, nil
}

func buildDayPicker(
	tm time.Time,
) ([][]models.InlineKeyboardButton, time.Time) {
	now := timeNow()

	if tm.Before(now) {
		tm = now.Round(30 * time.Minute)
	}

	var row [][]models.InlineKeyboardButton
	row = append(row,
		[]models.InlineKeyboardButton{
			{
				Text: tm.Weekday().String() + " " + tm.Format("02.01 15:04"),
				CopyText: models.CopyTextButton{
					Text: tm.Format(time.DateTime),
				},
			},
		})

	prevDay, nextDay := tm.AddDate(0, 0, -1), tm.AddDate(0, 0, 1)
	minusHour, plusHour := tm.Add(-time.Hour), tm.Add(time.Hour)
	minus30Mins, plus30Mins := tm.Add(-30*time.Minute), tm.Add(30*time.Minute)

	row = append(
		row,
		[]models.InlineKeyboardButton{
			{
				Text: "-1d",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(prevDay),
				),
			},
			{
				Text: "-1h",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(minusHour),
				),
			},

			{
				Text: "-30m",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(minus30Mins),
				),
			},
			{
				Text: "+30m",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(plus30Mins),
				),
			},
			{
				Text: "+1h",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(plusHour),
				),
			},
			{
				Text: "+1d",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(nextDay),
				),
			},
		},
	)

	return row, tm
}

func buildDurationPicker(
	state *tgbotv1.State_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) ([]models.InlineKeyboardButton, int32) {
	const (
		minHalves   = 2
		maxHalves   = 48
		valuesCount = 5
	)

	sel := cb.GetSelectDuration().GetHourHalfs()
	if sel == 0 {
		sel = state.GetDuration().GetHourHalfs()
	}

	if sel == 0 {
		sel = minHalves
	}

	duration := time.Duration(sel*_30mins) * time.Minute

	minus2Halves, minusHalf := sel-2, sel-1
	plusHalf, plus2Halves := sel+1, sel+2

	var row []models.InlineKeyboardButton
	if minus2Halves < minHalves {
		minus2Halves = minHalves
	}
	if minusHalf < minHalves {
		minusHalf = minHalves
	}
	if plusHalf > maxHalves {
		plusHalf = maxHalves
	}
	if plus2Halves > maxHalves {
		plus2Halves = maxHalves
	}

	row = append(row,
		models.InlineKeyboardButton{
			Text:         "-1h",
			CallbackData: encodeCallbackData(selectDurationCallback(minus2Halves)),
		},
		models.InlineKeyboardButton{
			Text:         "-30m",
			CallbackData: encodeCallbackData(selectDurationCallback(minusHalf)),
		},
		models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%.1fh", duration.Hours()),
			CallbackData: encodeCallbackData(selectDurationCallback(sel)),
		},
		models.InlineKeyboardButton{
			Text:         "+30m",
			CallbackData: encodeCallbackData(selectDurationCallback(plusHalf)),
		},
		models.InlineKeyboardButton{
			Text:         "+1h",
			CallbackData: encodeCallbackData(selectDurationCallback(plus2Halves)),
		},
	)
	return row, sel
}

func buildCheckCourtsResultsMessage(
	date time.Time,
	duration time.Duration,
	state *tgbotv1.State,
	pager *tgbotv1.Callbacks_Pager,
	clubs []*tgbotv1.State_AvailableCourts_Club,
) *UserMessage {

	baseMessage := fmt.Sprintf(
		"📅 %s\n🕘 %s\n🧭 %.1fh",
		date.Format("Monday 02.01"),
		date.Format("15:04"),
		duration.Hours(),
	)
	text, _ := messageTextWithState(baseMessage, state)

	return &UserMessage{
		Markdown: text,
		Keyboard: buildCheckCourtsResultsViewer(
			pager,
			clubs,
		),
	}
}

func buildCheckCourtsResultsViewer(
	pager *tgbotv1.Callbacks_Pager,
	clubs []*tgbotv1.State_AvailableCourts_Club,
) [][]models.InlineKeyboardButton {
	start, end := int(pager.GetOffset()), int(pager.GetOffset()+pager.GetLimit())
	if start < 0 {
		start = 0
	}
	if start > len(clubs) {
		start = len(clubs)
	}
	if end > len(clubs) {
		end = len(clubs)
	}
	clubsToShow := clubs[start:end]
	var pagerButtons []models.InlineKeyboardButton
	if start > 0 {
		pagerButtons = append(pagerButtons, models.InlineKeyboardButton{
			Text: "<",
			CallbackData: encodeCallbackData(updatePagerCallback(
				pager.GetLimit(), int32(start)-pager.GetLimit()),
			),
		})
	}

	if end < len(clubs) {
		pagerButtons = append(pagerButtons,
			models.InlineKeyboardButton{
				Text: ">",
				CallbackData: encodeCallbackData(
					updatePagerCallback(
						pager.GetLimit(), int32(end),
					),
				),
			},
		)
	}

	var kb [][]models.InlineKeyboardButton
	kb = append(kb, pagerButtons)
	for i, club := range clubsToShow {
		clubText := fmt.Sprintf("%s %s ", club.GetName(), club.GetAddress())
		kb = append(kb, []models.InlineKeyboardButton{
			{
				Text: fmt.Sprintf("%d. %s", start+i+1, clubText),
				URL:  fmt.Sprintf("%s", club.Url),
			},
		})
		for _, price := range club.GetPrices() {
			priceText := fmt.Sprintf("%s (%s)", price.GetAmount(), price.GetCourtType())
			kb = append(kb, []models.InlineKeyboardButton{
				{
					Text: priceText,
					CopyText: models.CopyTextButton{
						Text: fmt.Sprintf("%s %s", clubText, priceText),
					},
				},
			})
		}
	}

	return kb
}
