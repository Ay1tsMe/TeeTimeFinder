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
		assert.NoError(t, err, "Failed to write to temp config file")

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
