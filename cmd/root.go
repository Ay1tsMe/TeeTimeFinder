package cmd

import (
	"TeeTimeFinder/pkg/scraper"
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Global variables
var specifiedTime string
var specifiedDate string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "TeeTimeFinder",
	Short: "A CLI tool for finding golf tee times",
	Long:  `TeeTimeFinder is a CLI tool that allows you to find and book tee times for various golf courses.`,
	Run: func(cmd *cobra.Command, args []string) {
		runScraper() // Run the scraper logic when no subcommands are provided
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.TeeTimeFinder.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVarP(&specifiedTime, "time", "t", "", "Filter times within 1 hour before and after the specified time (e.g., 12:00)")
	rootCmd.PersistentFlags().StringVarP(&specifiedDate, "date", "d", "", "Specify the date for the tee time search (format: DD-MM)")
}

// Load the courses and URLs from config.txt
func loadCourses() (map[string]string, error) {
	courses := make(map[string]string)

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
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
			courses[courseName] = courseURL
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return courses, nil
}

func runScraper() {
	fmt.Println("Starting Golf Scraper...")

	courses, err := loadCourses()
	if err != nil {
		fmt.Printf("Error loading courses: %v\n", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// Prompt for the date if not provided
	var dateInput string
	if specifiedDate == "" {
		fmt.Print("Enter the date (DD-MM): ")
		dateInput, _ = reader.ReadString('\n')
		dateInput = strings.TrimSpace(dateInput)
	} else {
		dateInput = specifiedDate
		fmt.Printf("Using provided date: %s\n", dateInput)
	}

	// Parse the date input into day and month integers
	day, month, err := parseDayMonth(dateInput)
	if err != nil {
		fmt.Println("Invalid date format. Please use DD-MM.")
		return
	}
	fmt.Printf("Parsed day: %d, month: %d\n", day, month)

	// Prompt for time if not provided
	if specifiedTime == "" {
		// Ask for the time or allow the user to leave it blank
		fmt.Print("Enter the time (HH:MM) or press Enter to show all times: ")
		specifiedTime, _ = reader.ReadString('\n')
		specifiedTime = strings.TrimSpace(specifiedTime)

		if specifiedTime == "" {
			fmt.Println("No time specified, showing all times.")
		} else {
			// Parse the time if provided
			fmt.Printf("Using provided time: %s\n", specifiedTime)
		}
	} else {
		// Use the time provided via the flag
		fmt.Printf("Using provided time: %s\n", specifiedTime)
	}

	// Parse the specified time flag if provided
	var filterStartTime, filterEndTime time.Time
	if specifiedTime != "" {
		filterTime, err := time.Parse("15:04", specifiedTime)
		if err != nil {
			fmt.Println("Invalid time format. Please use HH:MM (24-hour format).")
			return
		}

		// Define the 1-hour range before and after the given time
		filterStartTime = filterTime.Add(-1 * time.Hour)
		filterEndTime = filterTime.Add(1 * time.Hour)
		fmt.Printf("Filtering results between %s and %s\n", filterStartTime.Format("15:04"), filterEndTime.Format("15:04"))
	}

	// Build a map of date indices to day and month
	dateIndex := -1
	for i := 0; i <= 4; i++ {
		date := time.Now().AddDate(0, 0, i)
		d := date.Day()
		m := int(date.Month())
		if d == day && m == month {
			dateIndex = i
			break
		}
	}

	if dateIndex == -1 {
		fmt.Println("Selected date is out of range (today to 5 days ahead).")
		return
	}

	// Categorise games
	var standardGames, promoGames []string
	gameToTimeslotURLs := make(map[string]map[string]string)

	// Iterate over the URLs and call the Scrape function
	for courseName, url := range courses {
		fmt.Printf("Scraping URL for course %s: %s\n", courseName, url)
		gameTimeslotURLs, err := scraper.ScrapeDates(url, dateIndex)
		if err != nil {
			fmt.Printf("Failed to scrape %s: %v\n", courseName, err)
			continue
		}

		// Categorise row names
		for name, timeslotURL := range gameTimeslotURLs {
			normalisedName := strings.TrimSpace(name)

			// Group all Twilight variations under "Twilight"
			if strings.Contains(strings.ToLower(normalisedName), "twilight") {
				normalisedName = "Twilight" // Normalise all "Twilight" variations to "Twilight"
			}
			// Group all public holiday variations
			if strings.Contains(strings.ToLower(normalisedName), "public holiday") {
				// Group variations of "9 holes" and "18 holes"
				if strings.Contains(strings.ToLower(normalisedName), "18 holes") || strings.Contains(strings.ToLower(normalisedName), "18 hole") {
					normalisedName = "18 Holes"
				} else if strings.Contains(strings.ToLower(normalisedName), "9 holes") || strings.Contains(strings.ToLower(normalisedName), "9 hole") {
					normalisedName = "9 Holes"
				}
			}

			// Group variations of "9 holes" and "18 holes"
			if strings.Contains(strings.ToLower(normalisedName), "18 holes") || strings.Contains(strings.ToLower(normalisedName), "18 hole") {
				normalisedName = "18 Holes"
			} else if strings.Contains(strings.ToLower(normalisedName), "9 holes") || strings.Contains(strings.ToLower(normalisedName), "9 hole") {
				normalisedName = "9 Holes"
			}

			fmt.Printf("Found available game: '%s'\n", normalisedName)

			// Categorise games
			if isStandardGame(normalisedName) {
				standardGames = append(standardGames, normalisedName)
			} else {
				promoGames = append(promoGames, normalisedName)
			}

			// Track the timeslot URLs and the course offering the game
			if gameToTimeslotURLs[normalisedName] == nil {
				gameToTimeslotURLs[normalisedName] = make(map[string]string)
			}
			gameToTimeslotURLs[normalisedName][courseName] = timeslotURL
		}
	}

	// Check if there are any available games
	if len(standardGames) == 0 && len(promoGames) == 0 {
		fmt.Println("No available games found on the selected date.")
		return
	}

	for {

		// Display standard games and promo option
		var gameOptions []string

		fmt.Println("\nSelect what game you want to play:")
		for i, game := range uniqueNames(standardGames) {
			fmt.Printf("%d. %s\n", i+1, game)
			gameOptions = append(gameOptions, game)
		}

		// Add "Promos" option
		if len(promoGames) > 0 {
			fmt.Printf("%d. Promos\n", len(gameOptions)+1)
			gameOptions = append(gameOptions, "Promos")
		}

		// Check if there are no game options
		if len(gameOptions) == 0 {
			fmt.Println("No available games to select.")
			return
		}

		// Read user input
		fmt.Print("Enter the number of your choice (or 'c' to cancel): ")
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		// Exit program if canceled
		if strings.ToLower(choiceStr) == "c" || strings.ToLower(choiceStr) == "cancel" {
			fmt.Println("Exiting TeeTimeFinder... Goodbye!")
			break
		}

		choice, err := strconv.Atoi(choiceStr)
		if err != nil || choice < 1 || choice > len(gameOptions) {
			fmt.Println("Invalid choice, please try again.")
			continue
		}

		selectedGame := gameOptions[choice-1]
		if selectedGame == "Promos" {
			// If "Promos" is selected, display promo games
			fmt.Println("\nSelect a promotional game:")
			for i, promo := range uniqueNames(promoGames) {
				fmt.Printf("%d. %s\n", i+1, promo)
			}
			fmt.Print("Enter the number of your choice (or 'c' to cancel): ")
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)

			// Exit program if canceled
			if strings.ToLower(choiceStr) == "c" || strings.ToLower(choiceStr) == "cancel" {
				fmt.Println("Exiting TeeTimeFinder... Goodbye!")
				break
			}

			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(promoGames) {
				fmt.Println("Invalid choice, please try again.")
				continue
			}
			selectedGame = promoGames[choice-1]
		}

		// Display courses that offer the selected game type
		fmt.Println("\nSelect a course that offers this game:")
		coursesForGame := gameToTimeslotURLs[selectedGame]
		var courseOptions []string
		for courseName := range coursesForGame {
			courseOptions = append(courseOptions, courseName)
		}

		for i, courseName := range courseOptions {
			fmt.Printf("%d. %s\n", i+1, courseName)
		}

		fmt.Print("Enter the number of your choice (or 'c' to cancel): ")
		choiceStr, _ = reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		// Exit program if canceled
		if strings.ToLower(choiceStr) == "c" || strings.ToLower(choiceStr) == "cancel" {
			fmt.Println("Exiting TeeTimeFinder... Goodbye!")
			break
		}

		choice, err = strconv.Atoi(choiceStr)
		if err != nil || choice < 1 || choice > len(courseOptions) {
			fmt.Println("Invalid choice, please try again.")
			continue
		}

		selectedCourse := courseOptions[choice-1]
		timeslotURL := coursesForGame[selectedCourse]

		fmt.Printf("\nYou selected: %s at %s\n", selectedGame, selectedCourse)

		availableTimes, err := scraper.ScrapeTimes(timeslotURL)
		if err != nil {
			fmt.Printf("Failed to scrape times for %s at %s: %v\n", selectedGame, selectedCourse, err)
			return
		}

		if len(availableTimes) == 0 {
			fmt.Printf("No available times found for %s at %s\n", selectedGame, selectedCourse)
		} else {
			// Store the times in a slice for sorting
			var times []string
			for t := range availableTimes {
				// Parse each time and filter based on -time flag if provided
				if specifiedTime != "" {
					gameTime, _ := time.Parse("03:04 pm", t)
					if gameTime.Before(filterStartTime) || gameTime.After(filterEndTime) {
						continue // Skip times outside the 1-hour range
					}
				}
				times = append(times, t)
			}

			// Check if any times remain after filtering
			if len(times) == 0 {
				fmt.Println("No games available at this time. Please try another hour.")
				return
			}

			type LayoutTime struct {
				Layout   string
				TimeSlot scraper.Timeslot
			}

			// Print the sorted times
			fmt.Println("Available times:")

			layoutTimes := make(map[string][]scraper.Timeslot)
			timeLayout := "03:04 pm"

			// First, group timeslots by layout
			for layout, timeslots := range availableTimes {
				for _, timeSlot := range timeslots {
					layoutTimes[layout] = append(layoutTimes[layout], timeSlot)
				}
			}

			// Sort the times within each layout
			for layout, timeslots := range layoutTimes {
				sort.Slice(timeslots, func(i, j int) bool {
					timeI, _ := time.Parse(timeLayout, timeslots[i].Time)
					timeJ, _ := time.Parse(timeLayout, timeslots[j].Time)
					return timeI.Before(timeJ)
				})
				layoutTimes[layout] = timeslots // Update with sorted times
			}

			// Now, sort the layouts by their earliest timeslot
			sortedLayouts := make([]string, 0, len(layoutTimes))
			for layout := range layoutTimes {
				sortedLayouts = append(sortedLayouts, layout)
			}

			sort.Slice(sortedLayouts, func(i, j int) bool {
				firstTimeI, _ := time.Parse(timeLayout, layoutTimes[sortedLayouts[i]][0].Time)
				firstTimeJ, _ := time.Parse(timeLayout, layoutTimes[sortedLayouts[j]][0].Time)
				return firstTimeI.Before(firstTimeJ)
			})

			// Print the sorted times by layout
			for _, layout := range sortedLayouts {
				fmt.Printf("\n%s:\n", layout)
				for _, timeSlot := range layoutTimes[layout] {
					fmt.Printf("%s: %d spots available\n", timeSlot.Time, timeSlot.AvailableSpots)
				}
			}

			// Ask the user if they want to book a game
			fmt.Print("Would you like to book a game at this course? (yes/no): ")
			bookingChoice, _ := reader.ReadString('\n')
			bookingChoice = strings.TrimSpace(strings.ToLower(bookingChoice))

			// Print the timeslot URL if they want to book
			if bookingChoice == "yes" || bookingChoice == "y" {
				fmt.Printf("Here is the URL for this game: %s\n", timeslotURL)
			} else {
				fmt.Println("Returning to course selection...")
			}

			// Go back to the selection menu after displaying the URL
		}
	}
}

// Helper function to parse the date input into day and month
func parseDayMonth(dateStr string) (int, int, error) {
	parts := strings.Split(dateStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("Invalid date format")
	}
	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return day, month, nil
}

// Helper function to check if a game is a standard game
func isStandardGame(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name)) // Trim spaces and convert to lowercase
	return name == "9 holes" || name == "18 holes" || name == "twilight"
}

// Helper function to get unique values from a slice
func uniqueNames(items []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range items {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
