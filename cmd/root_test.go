package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunScraper(t *testing.T) {
	// Redirect stdout to capture print statements
	var output bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Calculate a date that is two days after the current date
	currentDate := time.Now()
	twoDaysLater := currentDate.AddDate(0, 0, 2)
	specifiedDate = twoDaysLater.Format("02-01-2006")

	// Mock input for the time
	specifiedTime = "12:00"

	// Run the scraper function
	runScraper()

	// Capture the output
	w.Close()
	os.Stdout = oldStdout
	_, _ = output.ReadFrom(r)

	// Check if expected output is present
	outputStr := output.String()
	expectedText := "Starting Golf Scraper"
	if !strings.Contains(outputStr, expectedText) {
		t.Errorf("Expected output to contain '%s', but it didn't. Output: %s", expectedText, outputStr)
	}

	// Check for error message if no courses are loaded
	if strings.Contains(outputStr, "Error loading courses") {
		t.Errorf("Unexpected error while loading courses. Output: %s", outputStr)
	}
}

func TestHandleDateInput(t *testing.T) {
	// Test valid date input
	// Calculate a date that is two days after the current date
	currentDate := time.Now()
	twoDaysLater := currentDate.AddDate(0, 0, 2)
	specifiedDate = twoDaysLater.Format("02-01-2006")

	date, err := handleDateInput()
	if err != nil {
		t.Errorf("Unexpected error for valid date input: %v", err)
	}
	if date.Day() != twoDaysLater.Day() || date.Month() != twoDaysLater.Month() || date.Year() != twoDaysLater.Year() {
		t.Errorf("Expected date to be 15-11-2024, got %v", date)
	}

	// Test invalid date input
	specifiedDate = "invalid-date"
	_, err = handleDateInput()
	if err == nil {
		t.Error("Expected error for invalid date input, but got none")
	}
}

func TestHandleTimeInput(t *testing.T) {
	// Test valid time input
	specifiedTime = "12:30"
	startMinutes, endMinutes, err := handleTimeInput()
	if err != nil {
		t.Errorf("Unexpected error for valid time input: %v", err)
	}
	if startMinutes != 690 || endMinutes != 810 { // 690 = 11:30, 810 = 13:30
		t.Errorf("Expected time range to be 11:30 to 13:30, got %d to %d", startMinutes, endMinutes)
	}

	// Test invalid time input
	specifiedTime = "invalid-time"
	_, _, err = handleTimeInput()
	if err == nil {
		t.Error("Expected error for invalid time input, but got none")
	}
}

func TestLoadCourses(t *testing.T) {
	// Use a separate test config file to avoid overwriting the real one
	tempDir, err := ioutil.TempDir("", "teetimefinder_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	testConfigPath := tempDir + "/test_config.csv"

	// Create a temporary test config file
	file, err := os.Create(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer file.Close() // Close the file but do not delete it manually

	// Write mock data to the test file
	_, _ = file.WriteString("Wembley Golf Course,https://www.wembleygolf.com.au/guests/bookings/ViewPublicCalendar.msp\n")
	_, _ = file.WriteString("Fremantle Golf Course,https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000\n")

	// Load the courses using the test config path
	// Temporarily replace the configPath variable with the test config path
	originalConfigPath := configPath
	configPath = testConfigPath
	defer func() { configPath = originalConfigPath }() // Restore the original configPath

	courses, err := loadCourses()
	if err != nil {
		t.Errorf("Unexpected error loading courses: %v", err)
	}

	// Check if the courses are loaded correctly
	if len(courses) != 2 {
		t.Errorf("Expected 2 courses, got %d", len(courses))
	}
	if courses["Wembley Golf Course"] != "https://www.wembleygolf.com.au/guests/bookings/ViewPublicCalendar.msp" {
		t.Errorf("Course URL for Wembley Golf Course is incorrect")
	}
}
