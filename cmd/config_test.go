// cmd/config_test.go
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCreateDirAndConfigFile(t *testing.T) {
	// Setup: Use a test directory and config path
	originalConfigPath := configPath
	testDir := filepath.Join(os.TempDir(), "TeeTimeFinder_test")
	testConfigPath := filepath.Join(testDir, "config.txt")
	configPath = testConfigPath
	defer func() { configPath = originalConfigPath }()
	os.RemoveAll(testDir)

	// Ensure the directory does not exist
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatalf("Test directory should not exist before test")
	}

	// Test CreateDir
	if !CreateDir() {
		t.Fatal("CreateDir() returned false")
	}

	// Check if the directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatal("Expected directory to be created")
	}

	// Test ConfigExists when config file does not exist
	if ConfigExists() {
		t.Error("ConfigExists() should return false when config file does not exist")
	}

	// Create an empty config file
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	file.Close()

	// Test ConfigExists when config file exists
	if !ConfigExists() {
		t.Error("ConfigExists() should return true when config file exists")
	}

	// Cleanup
	os.RemoveAll(testDir)
}

func TestWriteInformationToConfigFile(t *testing.T) {
	// Setup
	originalConfigPath := configPath
	testDir := filepath.Join(os.TempDir(), "TeeTimeFinder_test")
	testConfigPath := filepath.Join(testDir, "config.txt")
	configPath = testConfigPath
	defer func() { configPath = originalConfigPath }()
	os.RemoveAll(testDir)

	// Ensure the directory exists
	if !CreateDir() {
		t.Fatal("Failed to create test directory")
	}

	// Courses to write
	courses := []string{
		"Course One,http://example.com/one",
		"Course Two,http://example.com/two",
	}

	// Test appendCoursesToFile
	err := appendCoursesToFile(courses)
	if err != nil {
		t.Fatalf("appendCoursesToFile failed: %v", err)
	}

	// Verify the content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expectedContent := strings.Join(courses, "\n") + "\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content:\n%s\nGot:\n%s", expectedContent, string(content))
	}

	// Cleanup
	os.RemoveAll(testDir)
}

func TestOverwriteConfigFile(t *testing.T) {
	// Setup
	originalConfigPath := configPath
	testDir := filepath.Join(os.TempDir(), "TeeTimeFinder_test")
	testConfigPath := filepath.Join(testDir, "config.txt")
	configPath = testConfigPath
	defer func() { configPath = originalConfigPath }()
	os.RemoveAll(testDir)

	// Ensure the directory exists
	if !CreateDir() {
		t.Fatal("Failed to create test directory")
	}

	// Initial courses
	initialCourses := []string{
		"Initial Course,http://example.com/initial",
	}
	err := appendCoursesToFile(initialCourses)
	if err != nil {
		t.Fatalf("Failed to write initial courses: %v", err)
	}

	// New courses to overwrite with
	newCourses := []string{
		"New Course One,http://example.com/newone",
		"New Course Two,http://example.com/newtwo",
	}

	// Test overwriteCoursesToFile
	err = overwriteCoursesToFile(newCourses)
	if err != nil {
		t.Fatalf("overwriteCoursesToFile failed: %v", err)
	}

	// Verify the content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expectedContent := strings.Join(newCourses, "\n") + "\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content after overwrite:\n%s\nGot:\n%s", expectedContent, string(content))
	}

	// Cleanup
	os.RemoveAll(testDir)
}

func TestEdgeCasesInConfigFile(t *testing.T) {
	// Edge cases to test
	tests := []struct {
		name            string
		sampleData      string
		expectedCourses map[string]string
	}{
		{
			name: "ValidData",
			sampleData: `Fremantle Golf Course,https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000
Point Walter Golf Course,https://pointwalter.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000
Collier Park Golf Course,https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000
Whaleback Golf Course,https://www.whalebackgolfcourse.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000
`,
			expectedCourses: map[string]string{
				"https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000":   "Fremantle Golf Course",
				"https://pointwalter.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000":       "Point Walter Golf Course",
				"https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000": "Collier Park Golf Course",
				"https://www.whalebackgolfcourse.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000":  "Whaleback Golf Course",
			},
		},
		{
			name: "DuplicateEntries",
			sampleData: `Duplicate Course,http://example.com/duplicate
Duplicate Course,http://example.com/duplicate
`,
			expectedCourses: map[string]string{
				"http://example.com/duplicate": "Duplicate Course",
			},
		},
		{
			name: "MalformedEntries",
			sampleData: `OnlyCourseName
,http://example.com/missingname
Valid Course,http://example.com/valid
InvalidEntryWithoutComma
`,
			expectedCourses: map[string]string{
				"http://example.com/valid": "Valid Course",
			},
		},
		{
			name: "EmptyLinesAndWhitespace",
			sampleData: `

   Course With Spaces   ,   http://example.com/spaces

Course Normal,http://example.com/normal

`,
			expectedCourses: map[string]string{
				"http://example.com/spaces": "Course With Spaces",
				"http://example.com/normal": "Course Normal",
			},
		},
		{
			name: "SpecialCharacters",
			sampleData: `Course & Co.,http://example.com/with?query=param&another=value
Cørse Ü,http://example.com/unicode
`,
			expectedCourses: map[string]string{
				"http://example.com/with?query=param&another=value": "Course & Co.",
				"http://example.com/unicode":                        "Cørse Ü",
			},
		},
		{
			name: "VeryLongEntries",
			sampleData: func() string {
				longCourseName := strings.Repeat("LongCourseName", 50)
				longURL := "http://example.com/" + strings.Repeat("path/", 50)
				return fmt.Sprintf("%s,%s\n", longCourseName, longURL)
			}(),
			expectedCourses: func() map[string]string {
				longCourseName := strings.Repeat("LongCourseName", 50)
				longURL := "http://example.com/" + strings.Repeat("path/", 50)
				return map[string]string{
					longURL: longCourseName,
				}
			}(),
		},
		{
			name: "InvalidURLs",
			sampleData: `Course InvalidURL,not a url
Course MissingParts,http:///missingparts
Course Empty,http://
`,
			expectedCourses: map[string]string{
				"not a url":            "Course InvalidURL",
				"http:///missingparts": "Course MissingParts",
				"http://":              "Course Empty",
			},
		},
		{
			name: "CommentedLines",
			sampleData: `# This is a comment
Course Active,http://example.com/active
#Course Commented,http://example.com/commented
`,
			expectedCourses: map[string]string{
				"http://example.com/active": "Course Active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			originalConfigPath := configPath
			testDir := filepath.Join(os.TempDir(), "TeeTimeFinder_test_"+tt.name)
			testConfigPath := filepath.Join(testDir, "config.txt")
			configPath = testConfigPath
			defer func() { configPath = originalConfigPath }()
			os.RemoveAll(testDir)

			// Ensure the directory exists
			if !CreateDir() {
				t.Fatal("Failed to create test directory")
			}

			// Write sample data to config file
			err := os.WriteFile(configPath, []byte(tt.sampleData), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			// Test loadExistingCourses
			courses := loadExistingCourses()
			if !reflect.DeepEqual(courses, tt.expectedCourses) {
				t.Errorf("Expected courses:\n%v\nGot:\n%v", tt.expectedCourses, courses)
			}

			// Cleanup
			os.RemoveAll(testDir)
		})
	}
}

func TestConfigCmdIntegration(t *testing.T) {
	// Setup
	originalConfigPath := configPath
	testDir := filepath.Join(os.TempDir(), "TeeTimeFinder_cmd_test")
	testConfigPath := filepath.Join(testDir, "config.txt")
	configPath = testConfigPath
	defer func() { configPath = originalConfigPath }()
	os.RemoveAll(testDir)

	// Ensure the directory exists
	if !CreateDir() {
		t.Fatal("Failed to create test directory")
	}

	// Simulate user inputs
	input := bytes.NewBufferString("Test Course\nhttp://example.com/test\nDone\n")

	// Create a pipe to simulate stdin
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Write input to the writer end of the pipe
	go func() {
		defer pw.Close()
		_, err := io.Copy(pw, input)
		if err != nil {
			t.Fatalf("Failed to write to pipe: %v", err)
		}
	}()

	// Backup original stdin
	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
		pr.Close()
	}()
	os.Stdin = pr

	// Set overwrite flag
	overwrite = true

	// Create a standalone command for testing
	testCmd := &cobra.Command{
		Use:   configCmd.Use,
		Short: configCmd.Short,
		Run:   configCmd.Run,
	}
	testCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite the existing config")

	// Execute the config command directly
	if err := testCmd.Execute(); err != nil {
		t.Fatalf("Failed to execute config command: %v", err)
	}

	// Verify the config file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expectedContent := "Test Course,http://example.com/test\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content:\n%s\nGot:\n%s", expectedContent, string(content))
	}

	// Cleanup
	os.RemoveAll(testDir)
}
