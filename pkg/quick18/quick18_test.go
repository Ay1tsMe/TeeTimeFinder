package quick18

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

func TestScrapeDates_Offline(t *testing.T) {
	t.Parallel()

	type fixture struct {
		name     string
		fileName string
	}

	cases := []fixture{
		{
			name:     "The Springs",
			fileName: "the_springs.html",
		},
		{
			name:     "Hamersley",
			fileName: "hamersley.html",
		},
	}

	// Snapshot date we expect to plug into teedate
	const snapshotDateISO = "2025-02-11"
	const expectedTeeDate = "20250211"

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Read local HTML snapshot
			fp := filepath.Join("testdata", c.fileName)
			html, err := os.ReadFile(fp)
			require.NoError(t, err, "failed to read local html file")

			// Parse snapshot date
			selectedDate, err := time.Parse("2006-01-02", snapshotDateISO)
			require.NoError(t, err)

			// Start http server to serve HTML file
			http_server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/teetimes/searchmatrix" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(html)
			}))
			defer http_server.Close()

			// Build a base URL pointing to the test server
			base, _ := url.Parse(http_server.URL)
			base.Path = "/teetimes/searchmatrix"

			// Run ScrapeDates
			results, scrapeErr := ScrapeDates(base.String(), selectedDate)

			assert.NoError(t, scrapeErr, "ScrapeDates should succeed against served snapshot")
			require.NotNil(t, results, "results map should not be nil")
			require.NotZero(t, len(results), "expected at least one available column/entry for snapshot")

			for header, ustr := range results {
				require.NotEmpty(t, header, "header (game type) should not be empty")

				u, perr := url.Parse(ustr)
				require.NoError(t, perr, "result URL should parse")

				// Scheme/host should match the test server
				assert.Equal(t, base.Scheme, u.Scheme)
				assert.Equal(t, base.Host, u.Host)
				assert.Contains(t, u.Path, "searchmatrix", "URL should point to Quick18 searchmatrix endpoint")

				q := u.Query()
				assert.Equal(t, expectedTeeDate, q.Get("teedate"), "teedate should match snapshot date (YYYYMMDD)")
			}
		})
	}
}
