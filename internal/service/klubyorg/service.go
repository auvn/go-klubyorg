package klubyorg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const _30minsDuration = 30 * time.Minute

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetCourts(
	ctx context.Context,
	ts time.Time,
	duration time.Duration,
) ([]CourtResult, error) {
	reservationDuration := duration.Minutes() / _30minsDuration.Minutes()
	body, err := query(ctx, SearchParams{
		Sport:          "1",
		Wojewodztwo:    "7",
		Miejscowosc:    "1",
		Dzielnica:      "0",
		Klub:           "0",
		DataRezerwacji: ts.Add(_30minsDuration).Round(_30minsDuration).Format("2006-01-02 15:04:05"),
		CzasRezerwacji: strconv.Itoa(int(reservationDuration)),
	})

	if err != nil {
		return nil, fmt.Errorf("query courts: %w", err)
	}

	defer body.Close()

	// parse the HTML
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("document from reader: %w", err)
	}

	var courts []CourtResult

	doc.Find("div").
		EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if id, _ := s.Attr("id"); id == "collapse_zajete" {
				return false
			}

			s.Find("table.table-bordered tbody tr").
				Each(func(i int, s *goquery.Selection) {

					// club
					club := s.Find("h3.list-group-item-heading a")
					address := s.Find("p.list-group-item-text")
					courtType := s.Find("td.vert-align h4").Eq(0)
					price := s.Find("td.vert-align h4").Eq(1)

					if club.Length() == 0 || price.Length() == 0 || price.Text() == "0,00" {
						return
					}

					href, _ := club.Attr("href")

					courts = append(courts, CourtResult{
						HRef:    _hrefBaseURL + href,
						Club:    simplifyString(club.Text()),
						Type:    simplifyString(courtType.Text()),
						Price:   simplifyString(price.Text()),
						Address: simplifyString(address.Text()),
					})
				})

			return true
		})

	if len(courts) == 0 {
		return nil, nil
	}

	slices.SortFunc(courts,
		func(a, b CourtResult) int {
			return strings.Compare(a.HRef, b.HRef)
		},
	)

	return courts, nil
	//
	// var last CourtResult
	// for _, c := range courts {
	// 	if last.HRef != c.HRef {
	// 		fmt.Printf(
	// 			"\n --- \t%s at %s\n\t%s\n",
	// 			c.Club, c.Address, c.HRef,
	// 		)
	// 	}
	// 	fmt.Printf(
	// 		"%s %s\n",
	// 		c.Price, c.Type,
	// 	)
	// 	last = c
	// }
}

const (
	_hrefBaseURL = "https://kluby.org/tenis/wyszukaj"
	_ajaxBaseURL = "https://kluby.org/ajax/wyszukiwarka.php"
)

type CourtResult struct {
	HRef    string
	Club    string
	Address string
	Type    string
	Price   string
}

func simplifyString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", "")
	return s
}

type SearchParams struct {
	Sport          string
	Wojewodztwo    string
	Miejscowosc    string
	Dzielnica      string
	Klub           string
	DataRezerwacji string // e.g. "2025-06-25 14:00:00"
	CzasRezerwacji string // e.g. "2"
}

func query(
	ctx context.Context,
	params SearchParams,
) (io.ReadCloser, error) {
	query := url.Values{}
	query.Set("dyscyplina", params.Sport)
	query.Set("wojewodztwo", params.Wojewodztwo)
	query.Set("miejscowosc", params.Miejscowosc)
	query.Set("dzielnica", params.Dzielnica)
	query.Set("klub", params.Klub)
	query.Set("data_rezerwacji", params.DataRezerwacji)
	query.Set("czas_rezerwacji", params.CzasRezerwacji)

	fullURL := _ajaxBaseURL + "?" + query.Encode()

	slog.InfoContext(ctx, "query", "url", fullURL)

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// if resp.Body != nil {
	// defer resp.Body.Close()
	// }

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// b, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return nil, fmt.Errorf("read all body: %w", err)
	// }

	// r := bytes.NewReader(b)
	// f, err := os.OpenFile("/tmp/response.html", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	// if err != nil {
	// 	return nil, fmt.Errorf("open file: %w", err)
	// }
	//
	// teeR := io.TeeReader(r, f)

	return resp.Body, nil
}
