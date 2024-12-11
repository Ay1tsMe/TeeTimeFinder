/*
Sets up TeeTimeFinder config file for the user.
Collects information such as:
- Golf Course Names
- URLs
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var configPath = filepath.Join(os.Getenv("HOME"), ".config", "TeeTimeFinder", "config.txt")
var overwrite bool

// Checks if the config file exists
func ConfigExists() bool {
	_, err := os.Stat(configPath)
	return err == nil
}

// Creates the .config/TeeTimeFinder directory if it doesn't exist
func CreateDir() bool {
	dir := filepath.Dir(configPath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Failed to create directory: %s\n", err)
			return false
		}
	}
	return true
}

// Loads the existing courses from the config file into a map
func loadExistingCourses() map[string]string {
	courses := make(map[string]string)

	if !ConfigExists() {
		return courses
	}

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("Failed to open config file: %s\n", err)
		return courses
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 {
			courseName := strings.TrimSpace(parts[0])
			courseURL := strings.TrimSpace(parts[1])
			courses[courseURL] = courseName
		}
	}
	return courses
}

// Appends new courses to the config file
func appendCoursesToFile(courses []string) error {
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, course := range courses {
		_, err := file.WriteString(course + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

// Overwrites the entire config file
func overwriteCoursesToFile(courses []string) error {
	file, err := os.OpenFile(configPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, course := range courses {
		_, err := file.WriteString(course + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure golf courses for TeeTimeFinder",
	Run: func(cmd *cobra.Command, args []string) {
		var courses []string
		reader := bufio.NewReader(os.Stdin)

		// Load existing courses if not overwriting
		existingCourses := map[string]string{}
		if !overwrite {
			existingCourses = loadExistingCourses()
		}

		fmt.Println("Please provide the golf course details.")
		for {
			// Get course name
			fmt.Print("Enter the name of the course (or 'done' to finish): ")
			courseName, _ := reader.ReadString('\n')
			courseName = strings.TrimSpace(courseName)

			if strings.ToLower(courseName) == "done" {
				break
			}

			// Get course URL
			fmt.Print("Enter the URL for the course: ")
			courseURL, _ := reader.ReadString('\n')
			courseURL = strings.TrimSpace(courseURL)

			// Validate course name and URL
			if courseName == "" || courseURL == "" {
				fmt.Println("Course name or URL cannot be empty. Please try again.")
				continue
			}

			// Check if the URL already exists
			if _, exists := existingCourses[courseURL]; exists {
				fmt.Println("Golf course already exists, skipping.")
				continue
			}

			// Add the course if it's not a duplicate
			courses = append(courses, fmt.Sprintf("%s,%s", courseName, courseURL))
			existingCourses[courseURL] = courseName
			fmt.Printf("%s has been added.\n", courseName)
		}

		// No courses to add
		if len(courses) == 0 {
			fmt.Println("No courses were added.")
			return
		}

		// Ensure the directory exists
		if !CreateDir() {
			return
		}

		// Either append or overwrite the file based on the -o flag
		var err error
		if overwrite {
			err = overwriteCoursesToFile(courses)
		} else {
			err = appendCoursesToFile(courses)
		}

		if err != nil {
			fmt.Printf("Failed to save to config file: %s\n", err)
			return
		}

		fmt.Println("Configuration saved!")
	},
}

// Command to show the config
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configured golf courses",
	Run: func(cmd *cobra.Command, args []string) {
		if !ConfigExists() {
			fmt.Println("No config file found. Please add courses using `TeeTimeFinder config`.")
			return
		}

		courses := loadExistingCourses()
		if len(courses) == 0 {
			fmt.Println("No courses found in the config.")
			return
		}

		fmt.Println("Configured Golf Courses:")
		i := 1

		for courseURL, courseName := range courses {
			fmt.Printf("%d) %s - %s\n", i, courseName, courseURL)
			i++
		}
	},
}

// Initializes the command and adds the -overwrite flag
func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite the existing config")

	configCmd.AddCommand(configShowCmd)
}
