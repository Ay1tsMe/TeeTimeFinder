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
			url:  "https://springs.quick18.com/teetimes/searchmatrix",
		},
		{
			name: "Hamersley",
			url:  "https://hamersley.quick18.com/teetimes/searchmatrix",
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

func TestScrapeTimes_Online(t *testing.T) {
	if !*runOnline {
		t.Skip("online test disabled; run with: go test -args -online")
	}

	type fixture struct {
		name        string
		calendarURL string
	}

	cases := []fixture{
		{
			name:        "The Springs",
			calendarURL: "https://springs.quick18.com/teetimes/searchmatrix?",
		},
		{
			name:        "Hamersley",
			calendarURL: "https://hamersley.quick18.com/teetimes/searchmatrix?",
		},
	}

	for _, cse := range cases {
		cse := cse
		t.Run(cse.name, func(t *testing.T) {
			t.Parallel()

			// Choose a future date to search
			selectedDate := time.Now().AddDate(0, 0, 3)

			// ScrapeDates to get one or more concrete URLs (with correct teedate, etc.)
			dateResults, err := ScrapeDates(cse.calendarURL, selectedDate)
			assert.NoError(t, err, "ScrapeDates should succeed against the live site")
			require.NotNil(t, dateResults, "dates results map should not be nil")

			if len(dateResults) == 0 {
				t.Logf("No columns with availability found for %s at %s (site may have no availability). Scraper still ran OK.",
					selectedDate.Format("2006-01-02"), cse.name)
				return
			}

			// Take one URL from the results
			var timesURL string
			for _, u := range dateResults {
				timesURL = u
				break
			}
			require.NotEmpty(t, timesURL, "expected at least one URL from ScrapeDates")

			// Sanity-check the URL structure before scraping times
			parsed, perr := url.Parse(timesURL)
			require.NoError(t, perr, "URL should parse correctly")
			assert.Contains(t, parsed.Path, "searchmatrix", "URL should point to the Quick18 searchmatrix endpoint")
			q := parsed.Query()
			assert.NotEmpty(t, q.Get("teedate"), "teedate should be present in query")

			// ScrapeTimes using the searchmatrix URL
			timeResults, scrapeErr := ScrapeTimes(timesURL)

			// Validate results
			assert.NoError(t, scrapeErr, "ScrapeTimes should succeed against the live page")
			require.NotNil(t, timeResults, "times results map should not be nil")

			// It’s possible no slots are available even if the page loads correctly.
			if len(timeResults) == 0 {
				t.Logf("No available tee times listed at %s for teedate=%s; scraper ran successfully.",
					cse.name, q.Get("teedate"))
				return
			}

			// Assertions
			for layout, slots := range timeResults {
				require.NotEmpty(t, layout, "layout (game type) should not be empty")
				require.NotEmpty(t, slots, "each layout should have at least one available slot")

				for _, slot := range slots {
					require.NotEmpty(t, slot.Time, "timeslot should include a time string")
					assert.Contains(t, slot.Time, ":", "time should look like HH:MM (contains :)")
					assert.Greater(t, slot.AvailableSpots, 0, "available spots should be > 0")
				}
			}
		})
	}
}

func TestScrapeTimes_Offline(t *testing.T) {
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

	for _, cse := range cases {
		cse := cse
		t.Run(cse.name, func(t *testing.T) {
			t.Parallel()

			// Load HTML snapshot
			fp := filepath.Join("testdata", cse.fileName)
			html, err := os.ReadFile(fp)
			require.NoError(t, err, "failed to read local html file")

			// Start http server to serve HTML file at the Quick18 times endpoint
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/teetimes/searchmatrix" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(html)
			}))
			defer srv.Close()

			// Build a base URL pointing to the test server's searchmatrix endpoint
			base, _ := url.Parse(srv.URL)
			base.Path = "/teetimes/searchmatrix"

			// Run ScrapeTimes against the served snapshot
			results, scrapeErr := ScrapeTimes(base.String())

			// Validate Results
			assert.NoError(t, scrapeErr, "ScrapeTimes should succeed against the served snapshot")
			require.NotNil(t, results, "results map should not be nil")
			require.NotZero(t, len(results), "expected at least one layout (game type) with available times")

			// Assertions
			for layout, slots := range results {
				require.NotEmpty(t, layout, "layout (game type) should not be empty")
				require.NotEmpty(t, slots, "each layout should have at least one available slot")

				for _, slot := range slots {
					// Time text formatting and non-empty
					require.NotEmpty(t, slot.Time, "timeslot should include a time string")
					assert.Contains(t, slot.Time, ":", "time should look like HH:MM (contains :)")

					// Only available slots should be included by ScrapeTimes
					assert.Greater(t, slot.AvailableSpots, 0, "available spots should be > 0")
				}
			}
		})
	}
}

func TestParseTimeCell(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Splits 2-line time to single line with space",
			in:   "2:30\nPM",
			want: "2:30 PM",
		},
		{
			name: "Trims and joins with lowercase am/pm preserved",
			in:   "  7:05\nam  ",
			want: "7:05 am",
		},
		{
			name: "No newline - returned as-is",
			in:   "10:00 PM",
			want: "10:00 PM",
		},
		{
			name: "Two lines with trailing newline",
			in:   "9:00\nAM\n",
			want: "9:00 AM",
		},
		{
			name: "Three lines collapses newlines to spaces",
			in:   "8:15\nAM\nExtra",
			want: "8:15 AM Extra",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := parseTimeCell(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestParsePlayers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want int
	}{
		{
			name: "Extract max players from range",
			in:   "1 to 4 players",
			want: 4,
		},
		{
			name: "Single Player",
			in:   "1 player",
			want: 1,
		},
		{
			name: "Up to N players",
			in:   "Up to 3 players",
			want: 3,
		},
		{
			name: "Hyphen range",
			in:   "2 - 4 players",
			want: 4,
		},
		{
			name: "No digits defaults to 1",
			in:   "No availability",
			want: 1,
		},
		{
			name: "Whitespace and suffix",
			in:   " 3 players ",
			want: 3,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := parsePlayers(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestNormaliseGameName(t *testing.T) {
	t.Parallel()

	type tc struct {
		name string
		in   string
		want string
	}
	cases := []tc{
		{
			name: "9 holes with allowed modifier in parentheses -> 9 Holes",
			in:   "9 Holes (Walking)",
			want: "9 Holes",
		},
		{
			name: "18 holes with allowed words -> 18 Holes",
			in:   "18 holes carts can be added",
			want: "18 Holes",
		},
		{
			name: "Promo with no hole count -> Title case",
			in:   "early bird special",
			want: "Early Bird Special",
		},
		{
			name: "9 hole midweek spacing/case -> 9 Holes",
			in:   "  9   hole  Midweek  ",
			want: "9 Holes",
		},
		{
			name: "18 holes with extra non-allowed words -> promo title",
			in:   "18 HOLES Twilight Special",
			want: "18 Holes Twilight Special",
		},
		{
			name: "Course name (allowed) + 9 holes -> 9 Holes",
			in:   "Maylands 9 Holes",
			want: "9 Holes",
		},
		{
			name: "Parentheses content removed and trimmed",
			in:   "9 Holes (carts)   ",
			want: "9 Holes",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := normaliseGameName(c.in)
			require.NotEmpty(t, got)
			assert.Equal(t, c.want, got)
		})
	}
}
