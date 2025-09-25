package cmd

import (
	"os"
	"path/filepath"
	"testing"

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
