package tg

import (
	"encoding/base64"
	"fmt"
	"time"

	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
	"google.golang.org/protobuf/proto"
)

func encodeCallbackData(cb *tgbotv1.Callbacks_Data) string {
	bb, err := proto.Marshal(cb)
	if err != nil {
		panic(err) // we are in trouble
	}

	return base64.RawURLEncoding.EncodeToString(bb)
}

func decodeCallbackData(str string) (*tgbotv1.Callbacks_Data, error) {
	bb, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	var p tgbotv1.Callbacks_Data
	if err := proto.Unmarshal(bb, &p); err != nil {
		return nil, fmt.Errorf("unmarshal proto: %w", err)
	}

	return &p, nil
}

func changeDateTimeCallback(
	newDate time.Time,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_ChangeDatetime{
			ChangeDatetime: &tgbotv1.Callbacks_ChangeDateTime{
				Datetime: int32(newDate.Unix()),
			},
		},
	}
}

func finalizeCallback() *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_Finalize{
			Finalize: &tgbotv1.Callbacks_Empty{},
		},
	}
}

func resetCallback() *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_Reset_{
			Reset_: &tgbotv1.Callbacks_Empty{},
		},
	}
}

func selectHourPrecisionCallback(
	halfes bool,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_SelectHourPrecision{
			SelectHourPrecision: &tgbotv1.Callbacks_SelectHourPrecision{
				Halfes: halfes,
			},
		},
	}
}

func selectHourCallback(
	selected int32,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_SelectHour{
			SelectHour: &tgbotv1.Callbacks_SelectHour{
				Hour: selected,
			},
		},
	}
}

func selectDateTimeCallback(
	selected time.Time,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_SelectDateTime{
			SelectDateTime: &tgbotv1.Callbacks_SelectDateTime{
				Datetime: int32(selected.Unix()),
			},
		},
	}
}

func selectDurationCallback(
	hourHalfs int32,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_SelectDuration{
			SelectDuration: &tgbotv1.Callbacks_SelectDuration{
				HourHalfs: hourHalfs,
			},
		},
	}
}

func checkCourtsState(
	ts time.Time,
	date time.Time,
	durationHourHalfs int32,
	hour int32,
) *tgbotv1.State {
	return &tgbotv1.State{
		Ts: int32(ts.Unix()),
		V: &tgbotv1.State_CheckCourts_{
			CheckCourts: &tgbotv1.State_CheckCourts{
				Datetime: &tgbotv1.Callbacks_SelectDateTime{
					Datetime: int32(date.Unix()),
				},
				Duration: &tgbotv1.Callbacks_SelectDuration{
					HourHalfs: durationHourHalfs,
				},
			},
		},
	}
}

func updatePagerCallback(
	limit, offset int32,
) *tgbotv1.Callbacks_Data {
	return &tgbotv1.Callbacks_Data{
		V: &tgbotv1.Callbacks_Data_UpdatePager{
			UpdatePager: &tgbotv1.Callbacks_Pager{
				Limit:  int32(limit),
				Offset: int32(offset),
			},
		},
	}
}
