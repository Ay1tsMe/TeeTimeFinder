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
	rootCmd.PersistentFlags().StringVarP(&specifiedDate, "date", "d", "", "Specify the date for the tee time search (format: DD-MM-YYYY)")
}

// Function to run the scraper
func runScraper() {
	fmt.Println("Starting Golf Scraper...")

	courses, err := loadCourses()
	if err != nil {
		fmt.Printf("Error loading courses: %v\n", err)
		return
	}

    selectedDate, err := handleDateInput()
    if err != nil {
        fmt.Println(err)
        return
    }

    filterStartMinutes, filterEndMinutes, err := handleTimeInput()
    if err != nil {
        fmt.Println(err)
        return
    }

	standardGames, promoGames, gameToTimeslotURLs := scrapeCourseData(courses, selectedDate)

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
		handleTimesDisplay(timeslotURL, selectedGame, selectedCourse, filterStartMinutes, filterEndMinutes)

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
func handleDateInput() (time.Time, error) {
    var dateInput string

    if specifiedDate == "" {
        fmt.Print("Enter the date (DD-MM-YYYY): ")
        dateInput = strings.TrimSpace(readInput())
    } else {
        dateInput = specifiedDate
        fmt.Printf("Using provided date: %s\n", dateInput)
    }

    // Parse the date
    selectedDate, err := time.Parse("02-01-2006", dateInput)
    if err != nil {
        return time.Time{}, fmt.Errorf("Invalid date format. Please use DD-MM-YYYY.")
    }

    if selectedDate.Before(time.Now()) {
        return time.Time{}, fmt.Errorf("Selected date is in the past.")
    }

    return selectedDate, nil
}

func handleTimeInput() (int, int, error) {
    var timeInput string

    if specifiedTime == "" {
        fmt.Print("Enter the time (HH:MM) or press Enter to show all times: ")
        timeInput = strings.TrimSpace(readInput())
    } else {
        timeInput = specifiedTime
        fmt.Printf("Using provided time: %s\n", timeInput)
    }

    if timeInput != "" {
        filterTimeMinutes, err := parseTimeToMinutes24(timeInput)
        if err != nil {
            return 0, 0, fmt.Errorf("Invalid time format. Please use HH:MM (24-hour format).")
        }
        filterStartMinutes := filterTimeMinutes - 60
        filterEndMinutes := filterTimeMinutes + 60
        fmt.Printf("Filtering results between %02d:%02d and %02d:%02d\n",
            filterStartMinutes/60, filterStartMinutes%60,
            filterEndMinutes/60, filterEndMinutes%60)
        return filterStartMinutes, filterEndMinutes, nil
    }

    return 0, 0, nil
}

func parseTimeToMinutes(timeStr string) (int, error) {
    timeStr = strings.TrimSpace(strings.ToUpper(timeStr))
    layouts := []string{"03:04 PM", "3:04 PM"}
    for _, layout := range layouts {
        t, err := time.Parse(layout, timeStr)
        if err == nil {
            return t.Hour()*60 + t.Minute(), nil
        }
    }
    fmt.Printf("Failed to parse timeStr '%s' with any known layout\n", timeStr)
    return 0, fmt.Errorf("failed to parse time '%s'", timeStr)
}

func parseTimeToMinutes24(timeStr string) (int, error) {
    t, err := time.Parse("15:04", timeStr)
    if err != nil {
        return 0, err
    }
    return t.Hour()*60 + t.Minute(), nil
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
func scrapeCourseData(courses map[string]string, selectedDate time.Time) ([]string, []string, map[string]map[string]string) {
    var standardGames, promoGames []string
    gameToTimeslotURLs := make(map[string]map[string]string)

    for courseName, url := range courses {
        fmt.Printf("Scraping URL for course %s: %s\n", courseName, url)
        gameTimeslotURLs, err := scraper.ScrapeDates(url, selectedDate)
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

func handleTimesDisplay(timeslotURL, selectedGame, selectedCourse string, filterStartMinutes, filterEndMinutes int) {
    availableTimes, err := scraper.ScrapeTimes(timeslotURL)
    if err != nil {
        fmt.Printf("Failed to scrape times for %s at %s: %v\n", selectedGame, selectedCourse, err)
        return
    }

    if len(availableTimes) == 0 {
        fmt.Printf("No available times found for %s at %s\n", selectedGame, selectedCourse)
        return
    }

    // Sort the times and layouts, filtering happens here
	sortedLayouts, layoutTimes := sortTimesByLayout(availableTimes, filterStartMinutes, filterEndMinutes)

    // Check if there are any times after filtering
    if len(sortedLayouts) == 0 {
        fmt.Println("No available times within the specified time range.")
        return
    }

    // Display the sorted times
    displaySortedTimes(layoutTimes, sortedLayouts)
}

func sortTimesByLayout(availableTimes map[string][]scraper.Timeslot, filterStartMinutes, filterEndMinutes int) ([]string, map[string][]scraper.Timeslot) {
    layoutTimes := make(map[string][]scraper.Timeslot)
    earliestTimes := make(map[string]int)

    for layout, timeslots := range availableTimes {
        for _, timeSlot := range timeslots {
            gameTimeMinutes, err := parseTimeToMinutes(timeSlot.Time)
            if err != nil {
                continue
            }

            // Apply filtering if specified time range is provided
            if filterStartMinutes != 0 || filterEndMinutes != 0 {
                if gameTimeMinutes < filterStartMinutes || gameTimeMinutes > filterEndMinutes {
                    continue
                }
            }

            layoutTimes[layout] = append(layoutTimes[layout], timeSlot)

            // Track the earliest time for each layout
            if earliestTime, exists := earliestTimes[layout]; !exists || gameTimeMinutes < earliestTime {
                earliestTimes[layout] = gameTimeMinutes
            }
        }

        // Sort times within each layout
        sort.Slice(layoutTimes[layout], func(i, j int) bool {
            timeIMinutes, _ := parseTimeToMinutes(layoutTimes[layout][i].Time)
            timeJMinutes, _ := parseTimeToMinutes(layoutTimes[layout][j].Time)
            return timeIMinutes < timeJMinutes
        })
    }

    // Sort layouts based on their earliest times
    sortedLayouts := make([]string, 0, len(earliestTimes))
    for layout := range layoutTimes {
        sortedLayouts = append(sortedLayouts, layout)
    }

    sort.Slice(sortedLayouts, func(i, j int) bool {
        return earliestTimes[sortedLayouts[i]] < earliestTimes[sortedLayouts[j]]
    })

    return sortedLayouts, layoutTimes
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
