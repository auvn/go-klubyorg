package tg

import (
	"github.com/auvn/go-klubyorg/internal/service/klubyorg"
	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
)

func collectAvailableCourts(
	results []klubyorg.CourtResult,
) *tgbotv1.State_AvailableCourts {
	index := map[string]*tgbotv1.State_AvailableCourts_Club{}
	var clubs []*tgbotv1.State_AvailableCourts_Club
	for _, c := range results {
		knownClub, ok := index[c.HRef]
		if !ok {
			newClub := &tgbotv1.State_AvailableCourts_Club{
				Name:    c.Club,
				Address: c.Address,
				Url:     c.HRef,
				Prices: []*tgbotv1.State_AvailableCourts_Price{
					{
						Amount:    c.Price.String(),
						CourtType: c.Type,
					},
				},
			}

			index[c.HRef] = newClub
			clubs = append(clubs, newClub)
		} else {
			knownClub.Prices = append(
				knownClub.Prices,
				&tgbotv1.State_AvailableCourts_Price{
					Amount:    c.Price.String(),
					CourtType: c.Type,
				},
			)
		}
	}

	return &tgbotv1.State_AvailableCourts{
		Clubs: clubs,
	}
}
