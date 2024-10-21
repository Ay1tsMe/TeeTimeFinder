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

var specifiedTime string
var specifiedDate string

var rootCmd = &cobra.Command{
	Use:   "TeeTimeFinder",
	Short: "A CLI tool for finding golf tee times",
	Long:  `TeeTimeFinder allows you to find and book tee times for MiClub golf courses.`,
	Run: func(cmd *cobra.Command, args []string) {
		runScraper()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&specifiedTime, "time", "t", "", "Filter times within 1 hour before and after the specified time (e.g., 12:00)")
	rootCmd.PersistentFlags().StringVarP(&specifiedDate, "date", "d", "", "Specify the date for the tee time search (format: DD-MM)")
}

// Function to run the scraper
func runScraper() {
	fmt.Println("Starting Golf Scraper...")

	courses, err := loadCourses()
	if err != nil {
		fmt.Printf("Error loading courses: %v\n", err)
		return
	}

	day, month, err := handleDateInput()
	if err != nil {
		fmt.Println(err)
		return
	}

	filterStartTime, filterEndTime, err := handleTimeInput()
	if err != nil {
		fmt.Println(err)
		return
	}

	dateIndex, err := calculateDateIndex(day, month)
	if err != nil {
		fmt.Println(err)
		return
	}

	standardGames, promoGames, gameToTimeslotURLs := scrapeCourseData(courses, dateIndex)

	if len(standardGames) == 0 && len(promoGames) == 0 {
		fmt.Println("No available games found on the selected date.")
		return
	}

	for {
		selectedGame := promptGameSelection(standardGames, promoGames, gameToTimeslotURLs)

		if selectedGame == "" {
			break
		}

		selectedCourse, timeslotURL := promptCourseSelection(gameToTimeslotURLs[selectedGame])

		if selectedCourse == "" {
			break
		}

		// Display the available times and prompt for booking
		handleTimesDisplay(timeslotURL, selectedGame, selectedCourse, filterStartTime, filterEndTime)

		// Ask user if they want to book this game
		fmt.Print("Would you like to book a game at this course? (yes/no): ")
		bookingChoice := strings.ToLower(strings.TrimSpace(readInput()))

		// Print the booking URL if they say yes
		if bookingChoice == "yes" || bookingChoice == "y" {
			fmt.Printf("Here is the URL for this game: %s\n", timeslotURL)
		} else {
			fmt.Println("Returning to game selection...")
		}
	}
}

// Function to load courses from config file
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

// Function to handle date input
func handleDateInput() (int, int, error) {
	reader := bufio.NewReader(os.Stdin)
	var dateInput string

	if specifiedDate == "" {
		fmt.Print("Enter the date (DD-MM): ")
		dateInput, _ = reader.ReadString('\n')
		dateInput = strings.TrimSpace(dateInput)
	} else {
		dateInput = specifiedDate
		fmt.Printf("Using provided date: %s\n", dateInput)
	}

	return parseDayMonth(dateInput)
}

// Function to calculate date index (for up to 5 days ahead)
func calculateDateIndex(day, month int) (int, error) {
	for i := 0; i <= 4; i++ {
		date := time.Now().AddDate(0, 0, i)
		if date.Day() == day && int(date.Month()) == month {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Selected date is out of range (today to 5 days ahead).")
}

// Function to handle time input
func handleTimeInput() (time.Time, time.Time, error) {
	reader := bufio.NewReader(os.Stdin)
	var filterStartTime, filterEndTime time.Time

	if specifiedTime == "" {
		fmt.Print("Enter the time (HH:MM) or press Enter to show all times: ")
		specifiedTime, _ = reader.ReadString('\n')
		specifiedTime = strings.TrimSpace(specifiedTime)
	}

	if specifiedTime != "" {
		filterTime, err := time.Parse("15:04", specifiedTime)
		if err != nil {
			return filterStartTime, filterEndTime, fmt.Errorf("Invalid time format. Please use HH:MM (24-hour format).")
		}
		filterStartTime = filterTime.Add(-1 * time.Hour)
		filterEndTime = filterTime.Add(1 * time.Hour)
		fmt.Printf("Filtering results between %s and %s\n", filterStartTime.Format("15:04"), filterEndTime.Format("15:04"))
	}

	return filterStartTime, filterEndTime, nil
}

// Function to parse day and month
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

// Function to scrape course data
func scrapeCourseData(courses map[string]string, dateIndex int) ([]string, []string, map[string]map[string]string) {
	var standardGames, promoGames []string
	gameToTimeslotURLs := make(map[string]map[string]string)

	for courseName, url := range courses {
		fmt.Printf("Scraping URL for course %s: %s\n", courseName, url)
		gameTimeslotURLs, err := scraper.ScrapeDates(url, dateIndex)
		if err != nil {
			fmt.Printf("Failed to scrape %s: %v\n", courseName, err)
			continue
		}

		standardGames, promoGames, gameToTimeslotURLs = categoriseGames(gameTimeslotURLs, courseName, standardGames, promoGames, gameToTimeslotURLs)
	}

	return standardGames, promoGames, gameToTimeslotURLs
}

// Function to categorise games and store them in maps
func categoriseGames(gameTimeslotURLs map[string]string, courseName string, standardGames, promoGames []string, gameToTimeslotURLs map[string]map[string]string) ([]string, []string, map[string]map[string]string) {
	for name, timeslotURL := range gameTimeslotURLs {
		normalisedName := normaliseGameName(name)
		fmt.Printf("DEBUG: Found available game: '%s' at course '%s'\n", normalisedName, courseName)

		// Check if it's a standard game
		if isStandardGame(normalisedName) {
			standardGames = append(standardGames, normalisedName)
		} else {
			promoGames = append(promoGames, normalisedName)
		}

		if gameToTimeslotURLs[normalisedName] == nil {
			gameToTimeslotURLs[normalisedName] = make(map[string]string)
		}
		gameToTimeslotURLs[normalisedName][courseName] = timeslotURL

		// Debug print to confirm timeslot URL is being added correctly
		fmt.Printf("DEBUG: Added timeslot URL for '%s' at '%s': %s\n", normalisedName, courseName, timeslotURL)
	}

	return standardGames, promoGames, gameToTimeslotURLs
}

// Function to normalise game names
func normaliseGameName(name string) string {
	normalised := strings.ToLower(strings.TrimSpace(name))
	if strings.Contains(normalised, "twilight") {
		return "Twilight"
	}
	if strings.Contains(normalised, "public holiday") {
		if strings.Contains(normalised, "18 holes") {
			return "18 Holes"
		} else if strings.Contains(normalised, "9 holes") {
			return "9 Holes"
		}
	}
	if strings.Contains(normalised, "18 holes") || strings.Contains(normalised, "18 hole") {
		return "18 Holes"
	}
	if strings.Contains(normalised, "9 holes") || strings.Contains(normalised, "9 hole") {
		return "9 Holes"
	}
	return name
}

// Function to display available games and handle game selection
func promptGameSelection(standardGames, promoGames []string, gameToTimeslotURLs map[string]map[string]string) string {
	var gameOptions []string
	fmt.Println("\nSelect what game you want to play:")
	for i, game := range uniqueNames(standardGames) {
		fmt.Printf("%d. %s\n", i+1, game)
		gameOptions = append(gameOptions, game)
	}

	if len(promoGames) > 0 {
		fmt.Printf("%d. Promos\n", len(gameOptions)+1)
		gameOptions = append(gameOptions, "Promos")
	}

	if len(gameOptions) == 0 {
		fmt.Println("No available games to select.")
		return ""
	}

	selectedGame := readChoice(gameOptions)

	if selectedGame == "Promos" {
		// Handle promo game selection
		if len(promoGames) > 1 {
			fmt.Println("\nSelect a promotional game:")
			for i, promo := range uniqueNames(promoGames) {
				fmt.Printf("%d. %s\n", i+1, promo)
			}
			selectedGame = readChoice(promoGames)
		} else {
			selectedGame = promoGames[0]
		}
	}

	return selectedGame
}

// Function to display available courses and handle course selection
func promptCourseSelection(coursesForGame map[string]string) (string, string) {
	if len(coursesForGame) == 0 {
		fmt.Println("No courses available for this promo.")
		return "", ""
	}

	var courseOptions []string
	for courseName := range coursesForGame {
		courseOptions = append(courseOptions, courseName)
	}

	fmt.Println("\nSelect a course that offers this game:")
	for i, courseName := range courseOptions {
		fmt.Printf("%d. %s\n", i+1, courseName)
	}

	selectedCourse := readChoice(courseOptions)
	return selectedCourse, coursesForGame[selectedCourse]
}

// Function to read user choice from a list of options
func readChoice(options []string) string {
	fmt.Print("Enter the number of your choice (or 'c' to cancel): ")
	choiceStr := strings.TrimSpace(readInput())

	if strings.ToLower(choiceStr) == "c" {
		return ""
	}

	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(options) {
		fmt.Println("Invalid choice, please try again.")
		return ""
	}

	return options[choice-1]
}

// Function to handle times display and sorting
func handleTimesDisplay(timeslotURL, selectedGame, selectedCourse string, filterStartTime, filterEndTime time.Time) {
	availableTimes, err := scraper.ScrapeTimes(timeslotURL)
	if err != nil {
		fmt.Printf("Failed to scrape times for %s at %s: %v\n", selectedGame, selectedCourse, err)
		return
	}

	if len(availableTimes) == 0 {
		fmt.Printf("No available times found for %s at %s\n", selectedGame, selectedCourse)
		return
	}

	// Sort the times and layouts
	sortedLayouts := sortTimesByLayout(availableTimes, filterStartTime, filterEndTime)

	// Display the sorted times
	displaySortedTimes(availableTimes, sortedLayouts)
}

// Function to sort times within each layout and then sort layouts by the earliest time
func sortTimesByLayout(availableTimes map[string][]scraper.Timeslot, filterStartTime, filterEndTime time.Time) []string {
	timeLayout := "03:04 pm"
	layoutTimes := make(map[string][]scraper.Timeslot)
	earliestTimes := make(map[string]time.Time)

	for layout, timeslots := range availableTimes {
		for _, timeSlot := range timeslots {
			gameTime, err := time.Parse(timeLayout, timeSlot.Time)
			if err != nil {
				continue
			}

			// Apply filtering if specified time range is provided
			if !filterStartTime.IsZero() && !filterEndTime.IsZero() {
				if gameTime.Before(filterStartTime) || gameTime.After(filterEndTime) {
					continue
				}
			}

			layoutTimes[layout] = append(layoutTimes[layout], timeSlot)

			// Track the earliest time for each layout
			if earliestTime, exists := earliestTimes[layout]; !exists || gameTime.Before(earliestTime) {
				earliestTimes[layout] = gameTime
			}
		}

		// Sort times within each layout
		sort.Slice(layoutTimes[layout], func(i, j int) bool {
			timeI, _ := time.Parse(timeLayout, layoutTimes[layout][i].Time)
			timeJ, _ := time.Parse(timeLayout, layoutTimes[layout][j].Time)
			return timeI.Before(timeJ)
		})
	}

	// Sort layouts based on their earliest times
	sortedLayouts := make([]string, 0, len(earliestTimes))
	for layout := range layoutTimes {
		sortedLayouts = append(sortedLayouts, layout)
	}

	sort.Slice(sortedLayouts, func(i, j int) bool {
		return earliestTimes[sortedLayouts[i]].Before(earliestTimes[sortedLayouts[j]])
	})

	return sortedLayouts
}

// Function to display sorted times
func displaySortedTimes(layoutTimes map[string][]scraper.Timeslot, sortedLayouts []string) {
	fmt.Println("Available times:")
	for _, layout := range sortedLayouts {
		fmt.Printf("\n%s:\n", layout)
		for _, timeSlot := range layoutTimes[layout] {
			fmt.Printf("%s: %d spots available\n", timeSlot.Time, timeSlot.AvailableSpots)
		}
	}
}

// Helper function to read user input
func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Helper function to check if a game is a standard game
func isStandardGame(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "9 holes" || name == "18 holes" || name == "twilight"
}

// Helper function to get unique names from a slice
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
