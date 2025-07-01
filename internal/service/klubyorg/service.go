package klubyorg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	_hrefBaseURL = "https://kluby.org/tenis/wyszukaj"
	_ajaxBaseURL = "https://kluby.org/ajax/wyszukiwarka.php"
)

const _30minsDuration = 30 * time.Minute

const (
	SportTennis = "1"
)

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
		Sport:          SportTennis,
		Wojewodztwo:    "7",
		Miejscowosc:    "1",
		Dzielnica:      "0",
		Klub:           "0",
		DataRezerwacji: ts.Format("2006-01-02 15:04:05"),
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

	doc.Find("table.table-bordered tbody tr").
		EachWithBreak(func(i int, s *goquery.Selection) bool {

			// club
			club := s.Find("h3.list-group-item-heading a")
			reserve := s.Find("td.vert-align a.btn")
			address := s.Find("p.list-group-item-text")
			data := s.Find("td.vert-align h4")
			isMuted := data.Parent().HasClass("text-muted")
			courtType := data.Eq(0)
			price := data.Eq(1)
			href, _ := reserve.Attr("href")

			if isMuted ||
				club.Length() == 0 ||
				href == "" ||
				price.Length() == 0 ||
				price.Text() == "0,00" {
				return true
			}

			priceStr := simplifyString(price.Text())
			parsedPrice, _, err := new(big.Float).Parse(
				strings.ReplaceAll(priceStr, ",", "."), 10,
			)
			if err != nil {
				return true
			}

			//nolint:errcheck
			res := CourtResult{
				HRef:    _hrefBaseURL + href,
				Club:    simplifyString(club.Text()),
				Type:    simplifyString(courtType.Text()),
				Price:   parsedPrice,
				Address: simplifyString(address.Text()),
			}
			courts = append(courts, res)
			return true
		})

	if len(courts) == 0 {
		return nil, nil
	}

	slices.SortFunc(courts,
		func(a, b CourtResult) int {
			priceCmp := a.Price.Cmp(b.Price)
			if priceCmp == 0 {
				return strings.Compare(a.HRef, b.HRef)
			}
			return priceCmp
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

type CourtResult struct {
	HRef    string
	Club    string
	Address string
	Type    string
	Price   *big.Float
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

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all body: %w", err)
	}

	r := bytes.NewReader(b)
	f, err := os.OpenFile("/tmp/response.html", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return io.NopCloser(io.TeeReader(r, f)), nil
}
