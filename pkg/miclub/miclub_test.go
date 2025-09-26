package miclub

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runOnline = flag.Bool("online", false, "run online tests that hit the live MiClub sites")

func TestScrapeDates_Online(t *testing.T) {
	if !*runOnline {
		t.Skip("online test disabled; run with: go test -args -online")
	}

	type fixture struct {
		name string
		url  string
	}

	cases := []fixture{
		{
			name: "Fremantle Public",
			url:  "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		},
		{
			name: "Collier Park",
			url:  "https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Choose a future date to search
			selectedDate := time.Now().AddDate(0, 0, 3)

			// Run ScrapeDates
			results, err := ScrapeDates(c.url, selectedDate)

			// Validate Results
			assert.NoError(t, err, "ScrapeDates should succeed against the live site")
			require.NotNil(t, results, "results map should not be nil")
			if len(results) == 0 {
				t.Logf("No timeslots found for %s (site may have no availability). Scraper still ran OK.", selectedDate.Format("2006-01-02"))
				return
			}

			// Assertions
			for row, ts := range results {
				require.NotEmpty(t, row, "row heading should not be empty")
				u, perr := url.Parse(ts)
				require.NoError(t, perr, "timeslot URL should parse correctly")

				assert.Contains(t, u.Path, "Timesheet", "timeslot URL should point to a timesheet endpoint")

				q := u.Query()
				assert.NotEmpty(t, q.Get("selectedDate"))
				assert.NotEmpty(t, q.Get("booking_resource_id"))
				assert.NotEmpty(t, q.Get("feeGroupId"), "fee_group_id should be present in query")

			}
		})
	}
}

func TestScrapeDates_Offline(t *testing.T) {
	t.Parallel()

	type fixture struct {
		name     string
		fileName string
	}

	cases := []fixture{
		{
			name:     "Fremantle Public",
			fileName: "fremantle_public_miclub.html",
		},
		{
			name:     "Collier Park",
			fileName: "collier_park_miclub.html",
		},
	}

	const bookingResourceID = "3000000"

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Load HTML snapshot
			fp := filepath.Join("testdata", c.fileName)
			html, err := os.ReadFile(fp)
			require.NoError(t, err, "failed to read local html file")

			// Use snapshot date of file
			selectedDate, err := time.Parse("2006-01-02", "2025-09-28")
			require.NoError(t, err)

			// Start http server to serve HTML file
			http_server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/guests/bookings/ViewPublicCalendar.msp" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(html)
			}))
			defer http_server.Close()

			// Build a base URL pointing to the test server
			base, _ := url.Parse(http_server.URL)
			base.Path = "/guests/bookings/ViewPublicCalendar.msp"
			q := base.Query()
			q.Set("booking_resource_id", bookingResourceID)
			base.RawQuery = q.Encode()

			// Run ScrapeDates
			results, scrapeErr := ScrapeDates(base.String(), selectedDate)

			// Validate Results
			assert.NoError(t, scrapeErr, "ScrapeDates should succeed against the served snapshot")
			require.NotNil(t, results, "results map should not be nil")
			require.NotZero(t, len(results), "expected at least one available timeslot for snapshot url")

			// Assertions
			for row, ts := range results {
				require.NotEmpty(t, row, "row heading should not be empty")
				u, err := url.Parse(ts)
				require.NoError(t, err, "timeslot URL should parse correctly")

				assert.Equal(t, base.Scheme, u.Scheme)
				assert.Equal(t, base.Host, u.Host)
				assert.Contains(t, u.Path, "ViewPublicTimesheet", "timeslot URL should point to a timesheet endpoint")

				q := u.Query()
				assert.Equal(t, "2025-09-27", q.Get("selectedDate"))
				assert.Equal(t, bookingResourceID, q.Get("booking_resource_id"))
				assert.NotEmpty(t, q.Get("feeGroupId"), "fee_group_id should be present in query")

			}
		})
	}
}
