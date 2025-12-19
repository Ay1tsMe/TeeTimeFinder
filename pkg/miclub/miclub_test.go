// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

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
			fileName: "fremantle_public_dates.html",
		},
		{
			name:     "Collier Park",
			fileName: "collier_park_dates.html",
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

func TestScrapeTimes_Online(t *testing.T) {
	if !*runOnline {
		t.Skip("online test disabled; run with: go test -args -online")
	}

	type fixture struct {
		name        string
		calendarURL string // ViewPublicCalendar; we'll derive a timesheet URL from this via ScrapeDates
	}

	cases := []fixture{
		{
			name:        "Fremantle Public",
			calendarURL: "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		},
		{
			name:        "Collier Park",
			calendarURL: "https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		},
	}

	for _, cse := range cases {
		cse := cse
		t.Run(cse.name, func(t *testing.T) {
			t.Parallel()

			// Choose a future date to search
			selectedDate := time.Now().AddDate(0, 0, 3)

			// ScrapeDates to get one or more timesheet URLs (with correct feeGroupId, selectedDate, etc.)
			dateResults, err := ScrapeDates(cse.calendarURL, selectedDate)
			assert.NoError(t, err, "ScrapeDates should succeed against the live site")
			require.NotNil(t, dateResults, "dates results map should not be nil")

			if len(dateResults) == 0 {
				t.Logf("No timesheet links found for %s at %s (site may have no availability). Scraper still ran OK.",
					selectedDate.Format("2006-01-02"), cse.name)
				return
			}

			// Get one timesheet URL from the results
			var timesheetURL string
			for _, u := range dateResults {
				timesheetURL = u
				break
			}
			require.NotEmpty(t, timesheetURL, "expected at least one timesheet URL from ScrapeDates")

			// Sanity-check the URL structure before scraping times
			parsed, perr := url.Parse(timesheetURL)
			require.NoError(t, perr, "timesheet URL should parse correctly")
			assert.Contains(t, parsed.Path, "ViewPublicTimesheet", "URL should point to a timesheet endpoint")
			q := parsed.Query()
			assert.NotEmpty(t, q.Get("selectedDate"), "selectedDate should be present in query")
			assert.NotEmpty(t, q.Get("booking_resource_id"), "booking_resource_id should be present in query")
			assert.NotEmpty(t, q.Get("feeGroupId"), "feeGroupId should be present in query")

			// ScrapeTimes using the public timesheet URL
			timeResults, scrapeErr := ScrapeTimes(timesheetURL)

			// Validate results
			assert.NoError(t, scrapeErr, "ScrapeTimes should succeed against the live timesheet")
			require.NotNil(t, timeResults, "times results map should not be nil")

			// It’s possible no slots are available even if the page loads correctly.
			if len(timeResults) == 0 {
				t.Logf("No available tee times listed at %s for %s; scraper ran successfully.",
					cse.name, q.Get("selectedDate"))
				return
			}

			// Assertions
			for layout, slots := range timeResults {
				require.NotEmpty(t, layout, "layout (course configuration) should not be empty")
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
			name:     "Fremantle Public",
			fileName: "fremantle_public_timesheet.html",
		},
		{
			name:     "Collier Park",
			fileName: "collier_park_timesheet.html",
		},
	}

	const (
		bookingResourceID = "3000000"
		feeGroupID        = "2000"
		selectedDate      = "2025-09-27"
	)

	for _, cse := range cases {
		cse := cse
		t.Run(cse.name, func(t *testing.T) {
			t.Parallel()

			// Load HTML snapshot
			fp := filepath.Join("testdata", cse.fileName)
			html, err := os.ReadFile(fp)
			require.NoError(t, err, "failed to read local html file")

			// Start http server to serve HTML file at the timesheet endpoint
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/guests/bookings/ViewPublicTimesheet.msp" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(html)
			}))
			defer srv.Close()

			// Build a base URL pointing to the test server's timesheet endpoint
			base, _ := url.Parse(srv.URL)
			base.Path = "/guests/bookings/ViewPublicTimesheet.msp"
			q := base.Query()
			q.Set("booking_resource_id", bookingResourceID)
			q.Set("feeGroupId", feeGroupID)
			q.Set("selectedDate", selectedDate)
			base.RawQuery = q.Encode()

			// Run ScrapeTimes against the served snapshot
			results, scrapeErr := ScrapeTimes(base.String())

			// Validate Results
			assert.NoError(t, scrapeErr, "ScrapeTimes should succeed against the served snapshot")
			require.NotNil(t, results, "results map should not be nil")
			require.NotZero(t, len(results), "expected at least one layout (course configuration) with available times")

			// Assertions
			for layout, slots := range results {
				require.NotEmpty(t, layout, "layout (course configuration) should not be empty")
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

func TestConstructTimeslotURL(t *testing.T) {
	t.Parallel()

	type fixture struct {
		name        string
		baseURL     string
		onclickAttr string
		wantFeeID   string
		wantDate    string
		wantValid   bool
	}

	cases := []fixture{
		{
			name:        "Valid onclick attribute builds correct URL",
			baseURL:     "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
			onclickAttr: "showTimeSheet('103961', '2025-09-30')",
			wantFeeID:   "103961",
			wantDate:    "2025-09-30",
			wantValid:   true,
		},
		{
			name:        "Extra whitespace and quotes handled correctly",
			baseURL:     "https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
			onclickAttr: "showTimesheet( '1500323733' , '2025-10-01' )",
			wantFeeID:   "1500323733",
			wantDate:    "2025-10-01",
			wantValid:   true,
		},
		{
			name:        "Malformed onclick returns empty string",
			baseURL:     "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp",
			onclickAttr: "invalidOnclick",
			wantValid:   false,
		},
		{
			name:        "Incorrect parameter count returns empty string",
			baseURL:     "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp",
			onclickAttr: "showTimesheet('103961')", // missing date
			wantValid:   false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			parsedBase, err := url.Parse(c.baseURL)
			require.NoError(t, err, "failed to parse base URL")

			result := constructTimeslotURL(parsedBase, c.onclickAttr)

			if !c.wantValid {
				assert.Empty(t, result, "expected empty result for invalid onclick input")
				return
			}

			require.NotEmpty(t, result, "expected a non-empty result URL")

			u, err := url.Parse(result)
			require.NoError(t, err, "constructed URL should parse successfully")

			// Check path correctness
			assert.Contains(t, u.Path, "ViewPublicTimesheet.msp", "path should be updated to timesheet endpoint")

			// Check query parameters
			q := u.Query()
			assert.Equal(t, c.wantFeeID, q.Get("feeGroupId"), "feeGroupId should match extracted param")
			assert.Equal(t, c.wantDate, q.Get("selectedDate"), "selectedDate should match extracted param")
			assert.Equal(t, "false", q.Get("weekends"), "weekends param should be set to false")

			// Booking resource ID (if present in base) should remain unchanged
			if parsedBase.Query().Get("booking_resource_id") != "" {
				assert.Equal(t, parsedBase.Query().Get("booking_resource_id"), q.Get("booking_resource_id"))
			}
		})
	}

}
