package tg

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	_30mins = 30

	_buttonCheckCourts = "Check Courts"
	_textCheckCourts   = "Select a date 📅 and duration 🕒 then hit the Check Courts button"
)

func (b *Bot) getState(
	msg *models.Message,
) (*tgbotv1.Callbacks_State, error) {
	if msg.ReplyMarkup == nil {
		return nil, nil
	}

	if len(msg.ReplyMarkup.InlineKeyboard) == 0 {
		return nil, nil
	}

	for _, row := range msg.ReplyMarkup.InlineKeyboard {
		for _, but := range row {
			if but.Text == _buttonCheckCourts { // TODO: const
				state, err := decodeCallbackData(but.CallbackData)
				if err != nil {
					return nil, fmt.Errorf("decode callback: %w", err)
				}

				switch {
				case state.GetFinalize() != nil:
					return state.GetFinalize(), nil
				}
			}
		}
	}

	return nil, nil
}

func (b *Bot) handleCheckCourtsCallbackQuery(
	ctx context.Context,
	state *tgbotv1.Callbacks_State,
	cbData *tgbotv1.Callbacks_Data,
	cb *models.CallbackQuery,
) error {
	ts := cbData.GetChangeDatetime().GetDatetime()
	if ts == 0 {
		ts = state.GetTs()
	}

	tm := timeUnix(int64(ts))

	slog.InfoContext(ctx, "state",
		"time", tm,
		"sate", state,
		"data", cbData,
	)

	if cmd := cbData.GetFinalize().GetCheckCourts(); cmd != nil {
		hourHalfs := cmd.GetDuration().GetHourHalfs()
		date := timeUnix(int64(cmd.GetDatetime().GetDatetime()))
		duration := time.Duration(hourHalfs*_30mins) * time.Minute
		result, err := b.courts.GetCourts(
			ctx,
			date,
			duration,
		)
		if err != nil {
			return fmt.Errorf("courts.GetCourts: %w", err)
		}

		type club struct {
			MainButton models.InlineKeyboardButton
			Options    []models.InlineKeyboardButton
		}
		byClub := map[string]*club{}
		buttons := [][]models.InlineKeyboardButton{}
		for _, c := range result {
			knownClub, ok := byClub[c.HRef]
			if !ok {
				byClub[c.HRef] = &club{
					MainButton: models.InlineKeyboardButton{
						Text: c.Club + " " + c.Address,
						URL:  c.HRef,
					},
					Options: []models.InlineKeyboardButton{
						{
							Text:     c.Price.String() + "",
							CopyText: models.CopyTextButton{},
						},
					},
				}
			} else {
				knownClub.Options = append(knownClub.Options, models.InlineKeyboardButton{
					Text:     c.Price.String() + "",
					CopyText: models.CopyTextButton{},
				})
			}
		}

		for _, c := range byClub {
			buttons = append(buttons, []models.InlineKeyboardButton{c.MainButton})
			buttons = append(buttons, c.Options)
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text:         "Back",
				CallbackData: encodeCallbackData(resetCallback(state)),
			},
		})
		_, err = b.me.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    cb.Message.Message.Chat.ID,
			MessageID: cb.Message.Message.ID,
			Text: fmt.Sprintf(
				"📅 %s 🕒 %.1fh",
				date.Format(time.DateTime), duration.Hours(),
			),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			},
		})
		return err
	}

	courtChecker, err := b.buildCourtChecker(
		cb.Message.Message.Chat.ID,
		tm,
		state.GetCheckCourts(),
		cbData,
	)
	if err != nil {
		return err
	}
	_, err = b.me.EditMessageText(ctx,
		&bot.EditMessageTextParams{
			ChatID:      cb.Message.Message.Chat.ID,
			MessageID:   cb.Message.Message.ID,
			Text:        _textCheckCourts,
			ReplyMarkup: courtChecker.ReplyMarkup,
		},
	)
	return err

}

func (b *Bot) buildCourtChecker(
	chatID int64,
	tm time.Time,
	state *tgbotv1.Callbacks_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) (*bot.SendMessageParams, error) {

	kb := [][]models.InlineKeyboardButton{}

	calendarPicker := buildDayPicker(tm)
	kb = append(kb, calendarPicker...)

	hourPicker, selectedTime := buildHourPicker(tm, state, cb)
	kb = append(kb, hourPicker)

	durationPicker, selectedDuration := buildDurationPicker(state, cb)
	kb = append(kb, durationPicker)

	kb = append(kb, []models.InlineKeyboardButton{
		{
			Text: _buttonCheckCourts,
			CallbackData: encodeCallbackData(
				finalizeCallback(
					checkCourtsState(
						tm,
						selectedTime,
						selectedDuration,
						0,
					),
				),
			),
		},
	})

	return &bot.SendMessageParams{
		ChatID: chatID,
		Text:   _textCheckCourts,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: kb,
		},
	}, nil
}

func buildDayPicker(
	tm time.Time,
) [][]models.InlineKeyboardButton {
	days := [][]models.InlineKeyboardButton{}
	now := timeNow().Truncate(24 * time.Hour)
	prevDate := tm.AddDate(0, 0, -1)
	if prevDate.Truncate(24 * time.Hour).Before(now) {
		prevDate = tm
	}

	days = append(days,
		[]models.InlineKeyboardButton{
			{
				Text: tm.Weekday().String() + " " + tm.Format("02.01"),
				CopyText: models.CopyTextButton{
					Text: tm.Format(time.DateTime),
				},
			},
		})

	days = append(
		days,
		[]models.InlineKeyboardButton{
			{
				Text: "<",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(prevDate),
				),
			},
			{
				Text: ">",
				CallbackData: encodeCallbackData(
					changeDateTimeCallback(tm.AddDate(0, 0, 1)),
				),
			},
		},
	)

	return days
}

func buildDurationPicker(
	state *tgbotv1.Callbacks_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) ([]models.InlineKeyboardButton, int32) {
	const (
		valuesCount = 5
	)

	sel := cb.GetSelectDuration().GetHourHalfs()
	if sel == 0 {
		sel = state.GetDuration().GetHourHalfs()
	}

	var values []models.InlineKeyboardButton
	for i := range valuesCount {
		halfes := (i + 1) * 2
		if i == 0 && sel == 0 {
			sel = int32(halfes)
		}

		text := fmt.Sprintf("%dm", halfes*_30mins)
		if sel == int32(halfes) {
			text = "*" + text
		}

		values = append(values, models.InlineKeyboardButton{
			Text:         text,
			CallbackData: encodeCallbackData(selectDurationCallback(int32(halfes))),
		})
	}

	return values, sel
}

// buildHourPicker
func buildHourPicker(
	tm time.Time,
	state *tgbotv1.Callbacks_CheckCourts,
	cb *tgbotv1.Callbacks_Data,
) ([]models.InlineKeyboardButton, time.Time) {
	const (
		valuesCount = 3
	)

	var row []models.InlineKeyboardButton
	now := timeNow()

	selectedHour := int(cb.GetSelectHour().GetHour())
	if selectedHour == 0 {
		dt := state.GetDatetime().GetDatetime()
		if dt > 0 {
			selectedHour = timeUnix(int64(dt)).Hour()
		}
	}

	start := tm
	if start.Truncate(time.Hour).Before(start) {
		start = start.Add(30 * time.Minute).Round(time.Hour)
	}
	startHour := start.Hour()

	prev, next := calculatePrevNextDates(start, valuesCount)

	if start.After(now) {
		row = append(row, models.InlineKeyboardButton{
			Text:         "<",
			CallbackData: encodeCallbackData(changeDateTimeCallback(prev)),
		})
	}

	var truncated bool
	for i := range valuesCount {
		hour := startHour + i
		if hour > _maxHour {
			truncated = true
			break
		}

		text := fmt.Sprintf("%d:00", hour)
		if (selectedHour == 0 && i == 0) || selectedHour == hour {
			text = "*" + text
			selectedHour = hour
		}

		row = append(row, models.InlineKeyboardButton{
			Text:         text,
			CallbackData: encodeCallbackData(selectHourCallback(int32(hour))),
		})
	}

	if !truncated {
		row = append(row,
			models.InlineKeyboardButton{
				Text:         ">",
				CallbackData: encodeCallbackData(changeDateTimeCallback(next)),
			},
		)
	}

	return row, time.Date(
		tm.Year(),
		tm.Month(),
		tm.Day(),
		selectedHour,
		0,
		0,
		0,
		_loc,
	)
}

func calculatePrevNextDates(
	tm time.Time,
	hoursShift int,
) (time.Time, time.Time) {

	nextHours := ensureMaxMin(tm.Hour()+hoursShift, _minHour, _maxHour)
	prevHours := ensureMaxMin(tm.Hour()-hoursShift, _minHour, _maxHour)

	next := time.Date(tm.Year(), tm.Month(), tm.Day(), nextHours, tm.Minute(), 0, 0, _loc)
	prev := time.Date(tm.Year(), tm.Month(), tm.Day(), prevHours, tm.Minute(), 0, 0, _loc)

	return prev, next
}

const (
	_minHour, _maxHour = 0, 23
)

func ensureMaxMin(v int, minv int, maxv int) int {
	if v < minv {
		return minv
	}

	if v >= maxv {
		return maxv
	}

	return v
}
