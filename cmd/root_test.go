package cmd

import (
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
		content := "Collier Park Golf Course,https://bookings.collierparkgolf.com.au\nHamersley Golf Course,https://hamersley.quick18.com/teetimes/searchmatrix"

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
		assert.Equal(t, "https://bookings.collierparkgolf.com.au", courses["Collier Park Golf Course"])
		assert.Equal(t, "https://hamersley.quick18.com/teetimes/searchmatrix", courses["Hamersley Golf Course"])
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
