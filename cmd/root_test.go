// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

package cmd

import (
	"TeeTimeFinder/pkg/shared"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadCourses(t *testing.T) {
	t.Run("Load Valid Courses", func(t *testing.T) {
		// Use temporary config file
		tmpDir := t.TempDir()
		tmpConfig := filepath.Join(tmpDir, "config.txt")
		content := "Collier Park Golf Course,https://bookings.collierparkgolf.com.au,miclub,false\nHamersley Golf Course,https://hamersley.quick18.com/teetimes/searchmatrix,quick18\n"

		err := os.WriteFile(tmpConfig, []byte(content), 0644)
		assert.NoError(t, err, "should be able to write temp config file")

		// Override global configPath temporarily
		originalConfigPath := configPath
		configPath = tmpConfig
		defer func() { configPath = originalConfigPath }()

		courses, err := loadCourses()

		// Assertions
		assert.NoError(t, err, "loadCourses() should not return an error")
		assert.Len(t, courses, 2, "should load exactly 2 courses")

		// Collier Park
		collier, ok := courses["Collier Park Golf Course"]
		assert.True(t, ok, "Collier Park should be present")
		assert.Equal(t, "https://bookings.collierparkgolf.com.au", collier.URL)
		assert.Equal(t, "miclub", collier.WebsiteType)
		assert.False(t, collier.Blacklisted)

		// Hamersley
		hamersley, ok := courses["Hamersley Golf Course"]
		assert.True(t, ok, "Hamersley should be present")
		assert.Equal(t, "https://hamersley.quick18.com/teetimes/searchmatrix", hamersley.URL)
		assert.Equal(t, "quick18", hamersley.WebsiteType)
		assert.False(t, hamersley.Blacklisted, "blacklisted should default to false when omitted")
	})

	t.Run("Missing Config File", func(t *testing.T) {
		// Set configPath to a non-existent file
		originalConfigPath := configPath
		configPath = "does_not_exist.txt"
		defer func() { configPath = originalConfigPath }()

		courses, err := loadCourses()

		// Assertions
		assert.Error(t, err, "expected error when config file does not exist")
		assert.Nil(t, courses, "courses map should be nil on error")
	})

}

func TestHandleDateInput(t *testing.T) {
	t.Run("Test Valid Date", func(t *testing.T) {
		// Save and restore global
		original := specifiedDate
		defer func() { specifiedDate = original }()

		// Generate a date 1 day in the future to pass the test
		futureDate := time.Now().AddDate(0, 0, 1).Format("02-01-2006")
		specifiedDate = futureDate

		selectedDate, err := handleDateInput()

		assert.NoError(t, err, "should not return error for a future date")
		assert.Equal(t, futureDate, selectedDate.Format("02-01-2006"), "returned date should match the input")
	})

	t.Run("Test Invalid Date Format", func(t *testing.T) {
		// Save and restore global
		original := specifiedDate
		defer func() { specifiedDate = original }()

		specifiedDate = "06/06/2024"

		_, err := handleDateInput()

		assert.Error(t, err, "should return error for invalid date")
	})

	t.Run("Test Date in the Past", func(t *testing.T) {
		// Save and restore global
		original := specifiedDate
		defer func() { specifiedDate = original }()

		specifiedDate = "17-08-2023"

		selectedDate, err := handleDateInput()

		assert.Error(t, err, "should return error for date in the past")
		assert.True(t, selectedDate.IsZero(), "date should be zero value on error")
	})
}

func TestHandleTimeInput(t *testing.T) {
	t.Run("Test Valid times", func(t *testing.T) {
		// Save and restore global
		original := specifiedTime
		defer func() { specifiedTime = original }()

		specifiedTime = "09:30"

		start, end, err := handleTimeInput()
		assert.NoError(t, err, "should be able to call function")

		// 09:30 -> 9*60 + 30 = 570
		// start = 570 - 60 = 510, end = 570 + 60 = 630
		assert.Equal(t, 510, start, "start time should equal 510")
		assert.Equal(t, 630, end, "end time should equal 630")
	})

	t.Run("Test Invalid times", func(t *testing.T) {
		// Save and restore global
		original := specifiedTime
		defer func() { specifiedTime = original }()

		specifiedTime = "25:99"

		start, end, err := handleTimeInput()

		assert.Error(t, err, "should return an error for invalid time format")
		assert.Equal(t, 0, start, "start time should be 0 on error")
		assert.Equal(t, 0, end, "start time should be 0 on error")
	})
}

func TestParseTimeToMinutes(t *testing.T) {
	t.Run("Valid 12-hour formats", func(t *testing.T) {
		response, err := parseTimeToMinutes("2:30 PM")
		assert.NoError(t, err)
		assert.Equal(t, 14*60+30, response, "2:30 PM should be 14:30")

		response, err = parseTimeToMinutes(" 7:05 am ")
		assert.NoError(t, err)
		assert.Equal(t, 7*60+5, response, "7:05 am should be 07:05")

		response, err = parseTimeToMinutes("10:00PM") // no space
		assert.NoError(t, err)
		assert.Equal(t, 22*60, response, "10:00PM should be 22:00")
	})

	t.Run("Invalid 12-hour format", func(t *testing.T) {
		_, err := parseTimeToMinutes("nope")
		assert.Error(t, err, "should error on invalid time")
	})
}

func TestParseTimeToMinutes24(t *testing.T) {
	t.Run("Valid 24-hour formats", func(t *testing.T) {
		response, err := parseTimeToMinutes24("09:30")
		assert.NoError(t, err)
		assert.Equal(t, 9*60+30, response)

		response, err = parseTimeToMinutes24("23:59")
		assert.NoError(t, err)
		assert.Equal(t, 23*60+59, response)
	})

	t.Run("Invalid 24-hour format", func(t *testing.T) {
		response, err := parseTimeToMinutes24("24:61")
		assert.Error(t, err)
		assert.Equal(t, 0, response)
	})
}

func TestFormatMinutesAs12Hour(t *testing.T) {
	cases := []struct {
		mins int
		want string
	}{
		{0, "12:00 AM"},
		{7*60 + 5, "07:05 AM"},
		{12 * 60, "12:00 PM"},
		{13*60 + 1, "01:01 PM"},
		{23*60 + 59, "11:59 PM"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			response := formatMinutesAs12Hour(c.mins)
			assert.Equal(t, c.want, response)
		})
	}
}

func TestFindCourseInsensitive(t *testing.T) {
	all := map[string]CourseConfig{
		"Hamersley Golf Course": {URL: "u1"},
		"Fremantle":             {URL: "u2"},
	}

	t.Run("Exact match", func(t *testing.T) {
		name, ok := findCourseInsensitive(all, "Hamersley Golf Course")
		assert.True(t, ok)
		assert.Equal(t, "Hamersley Golf Course", name)
	})

	t.Run("Case/space insensitive", func(t *testing.T) {
		name, ok := findCourseInsensitive(all, "  fremantle ")
		assert.True(t, ok)
		assert.Equal(t, "Fremantle", name)
	})

	t.Run("No match", func(t *testing.T) {
		_, ok := findCourseInsensitive(all, "Unknown")
		assert.False(t, ok)
	})
}

func TestIsStandardGame(t *testing.T) {
	t.Run("Standard names", func(t *testing.T) {
		assert.True(t, isStandardGame("9 Holes"))
		assert.True(t, isStandardGame("18 holes"))
		assert.True(t, isStandardGame(" twilight "))
	})

	t.Run("Non-standard names", func(t *testing.T) {
		assert.False(t, isStandardGame("Early Bird Special"))
		assert.False(t, isStandardGame("9 hole with carts")) // not exactly "9 holes"
	})
}

func TestUniqueNames(t *testing.T) {
	in := []string{"A", "B", "A", "C", "B", "D"}
	response := uniqueNames(in)
	assert.Equal(t, []string{"A", "B", "C", "D"}, response, "should dedupe while preserving first-seen order")
}

func TestNormaliseGameName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"9 holes with allowed modifier in parentheses -> 9 Holes", "9 Holes (Walking)", "9 Holes"},
		{"18 holes with allowed words -> 18 Holes", "18 holes carts can be added", "18 Holes"},
		{"Promo with no hole count -> Title case", "early bird special", "Early Bird Special"},
		{"9 hole midweek spacing/case -> 9 Holes", "  9   hole  Midweek  ", "9 Holes"},
		{"18 holes + extra words -> promo title", "18 HOLES Twilight Special", "18 Holes Twilight Special"},
		{"Course name (allowed) + 9 holes -> 9 Holes", "Maylands 9 Holes", "9 Holes"},
		{"Parentheses removed & trimmed", "9 Holes (carts)   ", "9 Holes"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			response := normaliseGameName(c.in)
			assert.Equal(t, c.want, response)
		})
	}
}

func TestSortLayoutsByEarliest(t *testing.T) {
	layoutTimes := map[string][]shared.TeeTimeSlot{
		"9 Holes":  {{Time: "10:00 AM", AvailableSpots: 4}},
		"18 Holes": {{Time: "8:30 AM", AvailableSpots: 4}},
		"Twilight": {{Time: "05:00 PM", AvailableSpots: 4}},
	}

	response := sortLayoutsByEarliest(layoutTimes)
	assert.Equal(t, []string{"18 Holes", "9 Holes", "Twilight"}, response)
}

func TestSortTimesByLayoutAndSpots(t *testing.T) {
	available := map[string][]shared.TeeTimeSlot{
		"18 Holes": {
			{Time: "9:00 AM", AvailableSpots: 2},
			{Time: "1:30 PM", AvailableSpots: 4},
			{Time: "8:00 AM", AvailableSpots: 1},
		},
		"9 Holes": {
			{Time: "7:15 AM", AvailableSpots: 4},
			{Time: "6:50 AM", AvailableSpots: 2},
		},
	}

	start := 8 * 60 // 08:00
	end := 14 * 60  // 14:00
	spots := 3      // need at least 3

	sortedLayouts, layoutTimes := sortTimesByLayoutAndSpots(available, start, end, spots)

	// Only "18 Holes" @ 1:30 PM should remain
	assert.Equal(t, []string{"18 Holes"}, sortedLayouts)
	assert.Len(t, layoutTimes, 1)
	assert.Len(t, layoutTimes["18 Holes"], 1)
	assert.Equal(t, "1:30 PM", layoutTimes["18 Holes"][0].Time)
	assert.Equal(t, 4, layoutTimes["18 Holes"][0].AvailableSpots)
}

func TestFilterAndSortTimes_NoFilters(t *testing.T) {
	available := map[string][]shared.TeeTimeSlot{
		"18 Holes": {
			{Time: "10:00 AM", AvailableSpots: 2},
			{Time: "08:30 AM", AvailableSpots: 2},
		},
		"9 Holes": {
			{Time: "07:15 AM", AvailableSpots: 4},
		},
	}

	response := filterAndSortTimes(available, 0, 0, 0)

	assert.Len(t, response, 2)

	// 18 Holes sorted 08:30 -> 10:00
	assert.Len(t, response["18 Holes"], 2)
	assert.Equal(t, "08:30 AM", response["18 Holes"][0].Time)
	assert.Equal(t, "10:00 AM", response["18 Holes"][1].Time)

	// 9 Holes unchanged
	assert.Len(t, response["9 Holes"], 1)
	assert.Equal(t, "07:15 AM", response["9 Holes"][0].Time)
}

func TestHandleSpotsInput(t *testing.T) {
	orig := specifiedSpots
	defer func() { specifiedSpots = orig }()

	t.Run("Zero -> no filter", func(t *testing.T) {
		specifiedSpots = 0
		used, err := handleSpotsInput()
		assert.NoError(t, err)
		assert.False(t, used)
	})

	t.Run("Valid 1..4 -> filter used", func(t *testing.T) {
		for _, v := range []int{1, 2, 3, 4} {
			specifiedSpots = v
			used, err := handleSpotsInput()
			assert.NoError(t, err)
			assert.True(t, used)
		}
	})

	t.Run("Invalid low/high", func(t *testing.T) {
		specifiedSpots = -1
		used, err := handleSpotsInput()
		assert.Error(t, err)
		assert.False(t, used)

		specifiedSpots = 5
		used, err = handleSpotsInput()
		assert.Error(t, err)
		assert.False(t, used)
	})
}
