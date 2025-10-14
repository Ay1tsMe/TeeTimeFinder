package quick18

import (
	"flag"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runOnline = flag.Bool("online", false, "run online tests that hit live Quick18 sites")

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
			name: "The Springs",
			url:  "https://springs.quick18.com/teetimes/searchmatrix?teedate=20251016",
		},
		{
			name: "Hamersley",
			url:  "https://hamersley.quick18.com/teetimes/searchmatrix?teedate=20251018",
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
			assert.NoError(t, err, "ScrapeDates should succeed against live Quick18 site")
			require.NotNil(t, results, "results map should not be nil")
			if len(results) == 0 {
				t.Logf("No timeslots found for %s (site may have no availability). Scraper still ran OK.", selectedDate.Format("2006-01-02"))
				return
			}

			wantTeeDate := selectedDate.Format("20060102")

			// Assertions
			for header, ustr := range results {
				require.NotEmpty(t, header, "header (game type) should not be empty")
				u, perr := url.Parse(ustr)
				require.NoError(t, perr, "result URL should parse")

				assert.Contains(t, u.Path, "searchmatrix", "URL should point to the Quick18 searchmatrix endpoint")

				q := u.Query()
				assert.Equal(t, wantTeeDate, q.Get("teedate"), "teedate should match selectedDate (YYYYMMDD)")
			}
		})
	}
}
