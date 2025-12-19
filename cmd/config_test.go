// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempConfigPath overrides the global configPath for the duration of a test.
// It returns the temp file path and a cleanup func that restores the original value.
func withTempConfigPath(t *testing.T, rel string) (string, func()) {
	t.Helper()
	tmp := t.TempDir()
	old := configPath
	cfg := filepath.Join(tmp, rel)
	configPath = cfg
	return cfg, func() { configPath = old }
}

func TestConfigExists(t *testing.T) {
	t.Run("Create missing directory", func(t *testing.T) {
		cfgPath, restore := withTempConfigPath(t, ".config/TeeTimeFinder/.config.txt")
		defer restore()

		// Ensure parent directory does not exist
		parent := filepath.Dir(cfgPath)
		_, statErr := os.Stat(parent)
		require.True(t, os.IsNotExist(statErr), "parent dir should not exist before CreateDir")

		ok := CreateDir()
		assert.True(t, ok, "CreateDir should return true on success")

		// Directory should now exist
		info, err := os.Stat(parent)
		require.NoError(t, err, "expected directory to exist after CreateDir")
		assert.True(t, info.IsDir(), "parent should be a directory")
	})

	t.Run("No operation when directory already exists", func(t *testing.T) {
		t.Parallel()

		cfgPath, restore := withTempConfigPath(t, ".config/TeeTimeFinder/.config.txt")
		defer restore()

		parent := filepath.Dir(cfgPath)
		require.NoError(t, os.MkdirAll(parent, 0o755), "pre-create parent directory")

		ok := CreateDir()
		assert.True(t, ok, "CreateDir should still return true if directory already exists")

		// Directory should still exist
		_, err := os.Stat(parent)
		assert.NoError(t, err, "directory should still exist")
	})
}

func TestLoadExistingConfig_NoFile(t *testing.T) {
	_, restore := withTempConfigPath(t, ".config/TeeTimeFinder/config.txt")
	defer restore()

	// Ensure file does not exist
	require.False(t, ConfigExists(), "config file should not exist for this test")

	response := loadExistingCourses()
	require.NotNil(t, response, "returned map should not be nil")
	assert.Empty(t, response, "expected empty map when no config file present")
}

// Load courses from testdata/config.txt
func TestLoadExistingCourses(t *testing.T) {
	// Point the global configPath at a real file in testdata.
	old := configPath
	configPath = filepath.Join("testdata", "config.txt")
	defer func() { configPath = old }()

	require.FileExists(t, configPath, "expected test config file in testdata/")

	response := loadExistingCourses()
	require.NotNil(t, response, "returned map should not be nil")

	// Expect exactly 4 valid entries keyed by URL
	require.Len(t, response, 4, "expected three parsed courses")

	// Collier Park
	collier, ok := response["https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000"]
	require.True(t, ok, "should contain Collier Park by URL key")
	assert.Equal(t, "Collier Park Golf Course", collier.Name)
	assert.Equal(t, "https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000", collier.URL)
	assert.Equal(t, "miclub", collier.WebsiteType)
	assert.True(t, collier.Blacklisted, "blacklist flag should be true for Collier Park")

	// Hamersley
	hamersley, ok := response["https://hamersley.quick18.com/teetimes/searchmatrix"]
	require.True(t, ok, "should contain Hamersley Golf Course")
	assert.Equal(t, "Hamersley Golf Course", hamersley.Name)
	assert.Equal(t, "Quick18", hamersley.WebsiteType)
	assert.False(t, hamersley.Blacklisted)

	// Fremantle
	freo, ok := response["https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000"]
	require.True(t, ok, "should contain Fremantle Golf Course")
	assert.Equal(t, "Fremantle Golf Course", freo.Name)
	assert.Equal(t, "miclub", freo.WebsiteType)
	assert.False(t, freo.Blacklisted)

	// The Springs
	springs, ok := response["https://springs.quick18.com/teetimes/searchmatrix"]
	require.True(t, ok, "should contain The Springs Golf Course")
	assert.Equal(t, "The Springs Golf Course", springs.Name)
	assert.Equal(t, "Quick18", springs.WebsiteType)
	assert.False(t, springs.Blacklisted)
}

func TestAppendCoursesToFile(t *testing.T) {
	cfgPath, restore := withTempConfigPath(t, ".config/TeeTimeFinder/config.txt")
	defer restore()

	require.True(t, CreateDir(), "CreateDir must succeed for test")

	// test cases
	courses := []CourseInfo{
		{
			Name:        "Fremantle Golf Course",
			URL:         "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
			WebsiteType: "miclub",
			Blacklisted: false,
		},
		{
			Name:        "The Springs Golf Course",
			URL:         "https://springs.quick18.com/teetimes/searchmatrix",
			WebsiteType: "Quick18",
			Blacklisted: false,
		},
	}

	// Append both courses to new file
	require.NoError(t, appendCoursesToFile(courses), "appendCoursesToFile should succeed")

	// Verify file content
	bytes, err := os.ReadFile(cfgPath)
	require.NoError(t, err, "expected config file to exist after append")
	expected := "" +
		"Fremantle Golf Course,https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000,miclub,false\n" +
		"The Springs Golf Course,https://springs.quick18.com/teetimes/searchmatrix,Quick18,false\n"
	assert.Equal(t, expected, string(bytes), "file content should match expected lines")

	// Verify results
	response := loadExistingCourses()
	require.Len(t, response, 2, "expected two courses parsed")

	freo := response["https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000"]
	assert.Equal(t, "Fremantle Golf Course", freo.Name)
	assert.Equal(t, "miclub", freo.WebsiteType)
	assert.False(t, freo.Blacklisted)

	springs := response["https://springs.quick18.com/teetimes/searchmatrix"]
	assert.Equal(t, "The Springs Golf Course", springs.Name)
	assert.Equal(t, "Quick18", springs.WebsiteType)
	assert.False(t, springs.Blacklisted)
}

func TestOverwriteCoursesToFile(t *testing.T) {
	cfgPath, restore := withTempConfigPath(t, ".config/TeeTimeFinder/config.txt")
	defer restore()

	require.True(t, CreateDir(), "CreateDir must succeed for test")

	// Seed the file with junk so we can prove it gets truncated.
	require.NoError(t, os.WriteFile(cfgPath, []byte("JUNK SHOULDN'T SURVIVE\n"), 0o644))

	// Overwrite with these two exact courses
	courses := []CourseInfo{
		{
			Name:        "Fremantle Golf Course",
			URL:         "https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
			WebsiteType: "miclub",
			Blacklisted: false,
		},
		{
			Name:        "The Springs Golf Course",
			URL:         "https://springs.quick18.com/teetimes/searchmatrix",
			WebsiteType: "Quick18",
			Blacklisted: false,
		},
	}
	require.NoError(t, overwriteCoursesToFile(courses), "overwriteCoursesToFile should truncate+write")

	// Verify file content is exactly the two lines
	gotBytes, err := os.ReadFile(cfgPath)
	require.NoError(t, err, "expected config file to exist after overwrite")
	expected := "" +
		"Fremantle Golf Course,https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000,miclub,false\n" +
		"The Springs Golf Course,https://springs.quick18.com/teetimes/searchmatrix,Quick18,false\n"
	assert.Equal(t, expected, string(gotBytes), "file content should match expected lines exactly")

	// Verify results
	response := loadExistingCourses()
	require.Len(t, response, 2, "expected two courses parsed after overwrite")

	freo := response["https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000"]
	require.NotNil(t, freo)
	assert.Equal(t, "Fremantle Golf Course", freo.Name)
	assert.Equal(t, "miclub", freo.WebsiteType)
	assert.False(t, freo.Blacklisted)

	springs := response["https://springs.quick18.com/teetimes/searchmatrix"]
	require.NotNil(t, springs)
	assert.Equal(t, "The Springs Golf Course", springs.Name)
	assert.Equal(t, "Quick18", springs.WebsiteType)
	assert.False(t, springs.Blacklisted)
}
