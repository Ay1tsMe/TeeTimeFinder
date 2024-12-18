package cmd

import (
	"TeeTimeFinder/pkg/scraper"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var allowedStandardModifiers = map[string]bool{
	"walking":  true,
	"midweek":  true,
	"carts":    true, // if "carts can be added" was inside parentheses, it’s already removed, but "carts" alone might remain
	"can":      true,
	"be":       true,
	"added":    true,
	"maylands": true,
}

var specifiedTime string
var specifiedDate string
var specifiedSpots int
var verboseMode bool
var courseList []string

// Pre-scraped data structure to hold all times if a time filter is used
var preScrapedTimes map[string]map[string]map[string][]scraper.Timeslot

var parenthesisRegex = regexp.MustCompile(`\(.+?\)`)
var nineHoleRegex = regexp.MustCompile(`\b9\s*hole(s)?\b`)
var eighteenHoleRegex = regexp.MustCompile(`\b18\s*hole(s)?\b`)

var rootCmd = &cobra.Command{
	Use:   "TeeTimeFinder",
	Short: "A CLI tool for finding golf tee times",
	Long:  `TeeTimeFinder allows you to find and book tee times for MiClub golf courses.`,
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runScraper(args)
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
	rootCmd.PersistentFlags().IntVarP(&specifiedSpots, "spots", "s", 0, "Filter timeslots based on available player spots (1-4)")
	rootCmd.PersistentFlags().StringArrayVarP(&courseList, "courses", "c", nil, "Specify particular courses to search")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "Enable verbose debug output")

	// Register the completion function here
	rootCmd.RegisterFlagCompletionFunc("courses", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Load courses from config
		courses, err := loadCourses()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Filter courses
		var completions []string
		for courseName := range courses {
			if strings.HasPrefix(strings.ToLower(courseName), strings.ToLower(toComplete)) {
				completions = append(completions, courseName)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	})
}

// Debug print functions that only print if verboseMode is true
func debugPrintln(a ...interface{}) {
	if verboseMode {
		fmt.Println("DEBUG:", fmt.Sprint(a...))
	}
}

func debugPrintf(format string, a ...interface{}) {
	if verboseMode {
		fmt.Printf("DEBUG: "+format, a...)
	}
}

// Function to run the scraper
func runScraper(args []string) {
	fmt.Println("Starting Golf Scraper...")

	courses, err := loadCourses()
	if err != nil {
		fmt.Printf("Error loading courses: %v\n", err)
		return
	}
	debugPrintf("Loaded courses: %+v\n", courses)

	var filtered map[string]string
	// We now check if the user provided any courses with the -c flag by checking courseList
	if len(courseList) > 0 {
		// The user specified one or more courses
		filtered = make(map[string]string)
		for _, cName := range courseList {
			cName = strings.TrimSpace(cName)
			url, exists := courses[cName]
			if !exists {
				fmt.Printf("Error: Course '%s' does not exist in config.\n", cName)
				return
			}
			filtered[cName] = url
		}
		courses = filtered
	} else {
		// No courses specified via -c, prompt the user as before
		fmt.Print("Press Enter to search ALL courses or type 'specify' to pick which courses to search: ")
		choice := strings.TrimSpace(readInput())

		if strings.ToLower(choice) == "specify" {
			filtered = make(map[string]string)
			fmt.Println("Enter course names line by line. Type 'done' when finished:")
			for {
				fmt.Print("Course Name: ")
				cName := strings.TrimSpace(readInput())
				if strings.ToLower(cName) == "done" {
					break
				}
				if cName == "" {
					fmt.Println("Please enter a valid course name or 'done' if finished.")
					continue
				}

				url, exists := courses[cName]
				if !exists {
					fmt.Printf("Error: Course '%s' does not exist in config.\n", cName)
					return
				}
				filtered[cName] = url
			}

			if len(filtered) == 0 {
				fmt.Println("No courses specified. Searching all courses.")
			} else {
				courses = filtered
			}
		} else if choice != "" {
			fmt.Println("Invalid choice. Searching all courses.")
		}
		// If user just pressed Enter, proceed with all courses
	}

	selectedDate, err := handleDateInput()
	if err != nil {
		fmt.Println(err)
		return
	}
	debugPrintf("Selected date: %s\n", selectedDate.Format("2006-01-02"))

	filterStartMinutes, filterEndMinutes, err := handleTimeInput()
	if err != nil {
		fmt.Println(err)
		return
	}
	debugPrintf("Time filters - start: %d, end: %d\n", filterStartMinutes, filterEndMinutes)

	spotsFilterUsed, err := handleSpotsInput()
	if err != nil {
		fmt.Println(err)
		return
	}
	debugPrintf("Spots filter used: %v, spots required: %d\n", spotsFilterUsed, specifiedSpots)

	standardGames, promoGames, gameToTimeslotURLs := scrapeCourseData(courses, selectedDate)

	debugPrintf("Standard Games: %v\n", standardGames)
	debugPrintf("Promo Games: %v\n", promoGames)

	if len(standardGames) == 0 && len(promoGames) == 0 {
		fmt.Println("No available games found on the selected date.")
		return
	}

	// If a specific time is given or spots, pre-scrape all times now.
	timeFilterUsed := (filterStartMinutes != 0 || filterEndMinutes != 0) || spotsFilterUsed

	if timeFilterUsed || spotsFilterUsed {
		fmt.Println("Filters specified. Searching all courses for available timeslots within the specified range. This can take some time, Please wait...")
		debugPrintln("Pre-scraping all times due to filters.")
		preScrapedTimes = preScrapeAllTimes(gameToTimeslotURLs, filterStartMinutes, filterEndMinutes, specifiedSpots)

		debugPrintln("Filtering available games and courses after pre-scrape.")
		standardGames, promoGames, gameToTimeslotURLs = filterAvailableGamesAndCourses(standardGames, promoGames, gameToTimeslotURLs, preScrapedTimes)

		debugPrintf("After filtering: StandardGames: %v, PromoGames: %v\n", standardGames, promoGames)
		if len(standardGames) == 0 && len(promoGames) == 0 {
			fmt.Println("No available games found for the specified time range.")
			return
		}
	}

	for {
		selectedGame := promptGameSelection(standardGames, promoGames, gameToTimeslotURLs)
		debugPrintf("User selected game: %s\n", selectedGame)

		if selectedGame == "" {
			debugPrintln("No game selected, stopping.")
			break
		}

		selectedCourse, timeslotURL := promptCourseSelection(gameToTimeslotURLs[selectedGame])
		debugPrintf("User selected course: %s, URL: %s\n", selectedCourse, timeslotURL)

		if selectedCourse == "" {
			debugPrintln("No course selected, stopping.")
			break
		}

		debugPrintf("Displaying times. timeFilterUsed: %v, spotsFilterUsed: %v\n", timeFilterUsed, spotsFilterUsed)
		if timeFilterUsed || spotsFilterUsed {
			handleTimesDisplayPreScraped(preScrapedTimes[selectedGame][selectedCourse], filterStartMinutes, filterEndMinutes, specifiedSpots)
		} else {
			handleTimesDisplay(timeslotURL, selectedGame, selectedCourse, filterStartMinutes, filterEndMinutes, specifiedSpots)
		}

		// Ask user if they want to book this game
		fmt.Print("Would you like to book a game at this course? (yes/no): ")
		bookingChoice := strings.ToLower(strings.TrimSpace(readInput()))

		if bookingChoice == "yes" || bookingChoice == "y" {
			fmt.Printf("Here is the URL for this game: %s\n", timeslotURL)
		} else {
			fmt.Println("Returning to game selection...")
		}
	}
}

func handleSpotsInput() (bool, error) {
	// If spots are specified via flag, use them
	if specifiedSpots != 0 {
		if specifiedSpots < 1 || specifiedSpots > 4 {
			return false, fmt.Errorf("Invalid spots value. Spots must be between 1 and 4.")
		}
		fmt.Printf("Using provided spots filter: %d\n", specifiedSpots)
		return true, nil
	}

	// If no spots flag is given, prompt the user
	fmt.Print("Enter the minimum number of spots (1-4) or press Enter to show all spots: ")
	input := strings.TrimSpace(readInput())
	if input == "" {
		return false, nil // No spots filter used
	}

	val, err := strconv.Atoi(input)
	if err != nil || val < 1 || val > 4 {
		fmt.Println("Invalid spots value. Please enter a number between 1 and 4.")
		return handleSpotsInput() // Recursively prompt again
	}

	specifiedSpots = val
	fmt.Printf("Filtering results for timeslots with at least %d spots.\n", specifiedSpots)
	return true, nil
}

// Function to pre-scrape all times if filters are specified
func preScrapeAllTimes(gameToTimeslotURLs map[string]map[string]string, filterStartMinutes, filterEndMinutes, spots int) map[string]map[string]map[string][]scraper.Timeslot {
	preScraped := make(map[string]map[string]map[string][]scraper.Timeslot)
	for game, courses := range gameToTimeslotURLs {
		debugPrintf("Pre-scrape: Checking game '%s'\n", game)
		if preScraped[game] == nil {
			preScraped[game] = make(map[string]map[string][]scraper.Timeslot)
		}
		for course, timeslotURL := range courses {
			debugPrintf("Pre-scrape: Scraping times for course '%s', URL: %s\n", course, timeslotURL)
			availableTimes, err := scraper.ScrapeTimes(timeslotURL)
			if err != nil {
				debugPrintf("Error scraping times for %s at %s: %v\n", game, course, err)
				continue
			}

			filteredTimes := filterAndSortTimes(availableTimes, filterStartMinutes, filterEndMinutes, spots)
			debugPrintf("Pre-scrape: '%s' at '%s' after filtering: %+v\n", game, course, filteredTimes)
			preScraped[game][course] = filteredTimes
		}
	}
	return preScraped
}

func filterAvailableGamesAndCourses(standardGames, promoGames []string, gameToTimeslotURLs map[string]map[string]string, preScraped map[string]map[string]map[string][]scraper.Timeslot) ([]string, []string, map[string]map[string]string) {
	newStandard := []string{}
	newPromo := []string{}

	for _, game := range standardGames {
		if courseMap, ok := preScraped[game]; ok {
			filteredCourseMap := make(map[string]string)
			for course, url := range gameToTimeslotURLs[game] {
				if len(courseMap[course]) > 0 {
					filteredCourseMap[course] = url
				} else {
					debugPrintf("Filtering out course '%s' for game '%s' - no times available.\n", course, game)
				}
			}
			if len(filteredCourseMap) > 0 {
				gameToTimeslotURLs[game] = filteredCourseMap
				newStandard = append(newStandard, game)
			} else {
				debugPrintf("Filtering out game '%s' from standardGames entirely (no courses left).\n", game)
				delete(gameToTimeslotURLs, game)
			}
		}
	}

	for _, game := range promoGames {
		if courseMap, ok := preScraped[game]; ok {
			filteredCourseMap := make(map[string]string)
			for course, url := range gameToTimeslotURLs[game] {
				if len(courseMap[course]) > 0 {
					filteredCourseMap[course] = url
				} else {
					debugPrintf("Filtering out course '%s' for promo game '%s' - no times available.\n", course, game)
				}
			}
			if len(filteredCourseMap) > 0 {
				gameToTimeslotURLs[game] = filteredCourseMap
				newPromo = append(newPromo, game)
			} else {
				debugPrintf("Filtering out promo game '%s' entirely (no courses left).\n", game)
				delete(gameToTimeslotURLs, game)
			}
		}
	}

	return newStandard, newPromo, gameToTimeslotURLs
}

func filterAndSortTimes(availableTimes map[string][]scraper.Timeslot, filterStartMinutes, filterEndMinutes, spots int) map[string][]scraper.Timeslot {
	debugPrintf("filterAndSortTimes called with start=%d, end=%d, spots=%d\n", filterStartMinutes, filterEndMinutes, spots)
	layoutTimes := make(map[string][]scraper.Timeslot)
	earliestTimes := make(map[string]int)

	for layout, timeslots := range availableTimes {
		debugPrintf("Layout '%s' before filtering: %v\n", layout, timeslots)
		for _, ts := range timeslots {
			gameTimeMinutes, err := parseTimeToMinutes(ts.Time)
			if err != nil {
				debugPrintf("Time parse error for '%s': %v\n", ts.Time, err)
				continue
			}

			if (filterStartMinutes != 0 || filterEndMinutes != 0) &&
				(gameTimeMinutes < filterStartMinutes || gameTimeMinutes > filterEndMinutes) {
				continue
			}

			if spots > 0 && ts.AvailableSpots < spots {
				continue
			}

			layoutTimes[layout] = append(layoutTimes[layout], ts)
			if earliestTime, exists := earliestTimes[layout]; !exists || gameTimeMinutes < earliestTime {
				earliestTimes[layout] = gameTimeMinutes
			}
		}

		sort.Slice(layoutTimes[layout], func(i, j int) bool {
			timeIMinutes, _ := parseTimeToMinutes(layoutTimes[layout][i].Time)
			timeJMinutes, _ := parseTimeToMinutes(layoutTimes[layout][j].Time)
			return timeIMinutes < timeJMinutes
		})
		debugPrintf("Layout '%s' after filtering: %v\n", layout, layoutTimes[layout])
	}

	return layoutTimes
}

func handleTimesDisplayPreScraped(layoutTimes map[string][]scraper.Timeslot, filterStartMinutes, filterEndMinutes, spots int) {
	debugPrintf("handleTimesDisplayPreScraped called with layouts: %v\n", layoutTimes)
	if len(layoutTimes) == 0 {
		fmt.Println("No available times with the specified filters.")
		return
	}
	displaySortedTimes(layoutTimes, sortLayoutsByEarliest(layoutTimes))
}

func sortLayoutsByEarliest(layoutTimes map[string][]scraper.Timeslot) []string {
	earliestTimes := make(map[string]int)
	for layout, times := range layoutTimes {
		if len(times) > 0 {
			mins, _ := parseTimeToMinutes(times[0].Time)
			earliestTimes[layout] = mins
		}
	}

	sortedLayouts := make([]string, 0, len(earliestTimes))
	for layout := range earliestTimes {
		sortedLayouts = append(sortedLayouts, layout)
	}

	sort.Slice(sortedLayouts, func(i, j int) bool {
		return earliestTimes[sortedLayouts[i]] < earliestTimes[sortedLayouts[j]]
	})
	return sortedLayouts
}

func handleTimesDisplay(timeslotURL, selectedGame, selectedCourse string, filterStartMinutes, filterEndMinutes, spots int) {
	debugPrintf("handleTimesDisplay for %s at %s, URL: %s\n", selectedGame, selectedCourse, timeslotURL)
	availableTimes, err := scraper.ScrapeTimes(timeslotURL)
	if err != nil {
		fmt.Printf("Failed to scrape times for %s at %s: %v\n", selectedGame, selectedCourse, err)
		return
	}

	if len(availableTimes) == 0 {
		fmt.Printf("No available times found for %s at %s\n", selectedGame, selectedCourse)
		return
	}

	sortedLayouts, layoutTimes := sortTimesByLayoutAndSpots(availableTimes, filterStartMinutes, filterEndMinutes, spots)

	if len(sortedLayouts) == 0 {
		fmt.Println("No available times with the specified filters.")
		return
	}

	displaySortedTimes(layoutTimes, sortedLayouts)
}

func sortTimesByLayoutAndSpots(availableTimes map[string][]scraper.Timeslot, filterStartMinutes, filterEndMinutes, spots int) ([]string, map[string][]scraper.Timeslot) {
	debugPrintf("sortTimesByLayoutAndSpots called with availableTimes: %v\n", availableTimes)
	layoutTimes := make(map[string][]scraper.Timeslot)
	earliestTimes := make(map[string]int)

	for layout, timeslots := range availableTimes {
		for _, timeSlot := range timeslots {
			gameTimeMinutes, err := parseTimeToMinutes(timeSlot.Time)
			if err != nil {
				debugPrintf("Time parse error '%s': %v\n", timeSlot.Time, err)
				continue
			}

			if (filterStartMinutes != 0 || filterEndMinutes != 0) &&
				(gameTimeMinutes < filterStartMinutes || gameTimeMinutes > filterEndMinutes) {
				continue
			}

			if spots > 0 && timeSlot.AvailableSpots < spots {
				continue
			}

			layoutTimes[layout] = append(layoutTimes[layout], timeSlot)
			if earliestTime, exists := earliestTimes[layout]; !exists || gameTimeMinutes < earliestTime {
				earliestTimes[layout] = gameTimeMinutes
			}
		}

		sort.Slice(layoutTimes[layout], func(i, j int) bool {
			timeIMinutes, _ := parseTimeToMinutes(layoutTimes[layout][i].Time)
			timeJMinutes, _ := parseTimeToMinutes(layoutTimes[layout][j].Time)
			return timeIMinutes < timeJMinutes
		})
		debugPrintf("Layout '%s' after sorting in sortTimesByLayoutAndSpots: %v\n", layout, layoutTimes[layout])
	}

	sortedLayouts := make([]string, 0, len(earliestTimes))
	for layout := range layoutTimes {
		sortedLayouts = append(sortedLayouts, layout)
	}

	sort.Slice(sortedLayouts, func(i, j int) bool {
		return earliestTimes[sortedLayouts[i]] < earliestTimes[sortedLayouts[j]]
	})

	return sortedLayouts, layoutTimes
}

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

func handleDateInput() (time.Time, error) {
	var dateInput string

	if specifiedDate == "" {
		fmt.Print("Enter the date (DD-MM-YYYY): ")
		dateInput = strings.TrimSpace(readInput())
	} else {
		dateInput = specifiedDate
		fmt.Printf("Using provided date: %s\n", dateInput)
	}

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

func categoriseGames(gameTimeslotURLs map[string]string, courseName string, standardGames, promoGames []string, gameToTimeslotURLs map[string]map[string]string) ([]string, []string, map[string]map[string]string) {
	for name, timeslotURL := range gameTimeslotURLs {
		debugPrintf("Categorising game: '%s'\n", name)
		normalisedName := normaliseGameName(name)
		debugPrintf("Normalised game name '%s' to '%s'\n", name, normalisedName)

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

func normaliseGameName(originalName string) string {
	// Lowercase and trim
	name := strings.ToLower(strings.TrimSpace(originalName))
	// Remove parentheses and their content
	name = parenthesisRegex.ReplaceAllString(name, "")
	name = strings.TrimSpace(name)

	// Check if it contains 9 or 18 hole references
	hasNine := strings.Contains(name, "9 hole")
	hasEighteen := strings.Contains(name, "18 hole")

	// If no hole count found, it's a promo
	if !hasNine && !hasEighteen {
		return strings.Title(name)
	}

	// Use regex to safely replace "9 hole(s)" with "9 holes"
	name = nineHoleRegex.ReplaceAllString(name, "9 holes")
	// Use regex to safely replace "18 hole(s)" with "18 holes"
	name = eighteenHoleRegex.ReplaceAllString(name, "18 holes")

	// Split into words
	words := strings.Fields(name)

	// Remove the "9 holes" or "18 holes" from words
	filtered := []string{}
	skipNext := false
	for i, w := range words {
		if w == "9" && i+1 < len(words) && words[i+1] == "holes" {
			skipNext = true
			continue
		}
		if w == "18" && i+1 < len(words) && words[i+1] == "holes" {
			skipNext = true
			continue
		}
		if skipNext {
			skipNext = false
			continue
		}
		filtered = append(filtered, w)
	}

	// Remove allowed standard modifiers
	finalWords := []string{}
	for _, w := range filtered {
		if !allowedStandardModifiers[w] {
			finalWords = append(finalWords, w)
		}
	}

	// If no extra words remain, it's a pure standard game
	if len(finalWords) == 0 {
		if hasNine {
			return "9 Holes"
		}
		if hasEighteen {
			return "18 Holes"
		}
	}

	// Otherwise, it's a promo
	return strings.Title(name)
}

func promptGameSelection(standardGames, promoGames []string, gameToTimeslotURLs map[string]map[string]string) string {
	fmt.Println("\nSelect what game you want to play:")
	gameOptions := []string{}
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

func sortTimesByLayout(availableTimes map[string][]scraper.Timeslot, filterStartMinutes, filterEndMinutes int) ([]string, map[string][]scraper.Timeslot) {
	layoutTimes := make(map[string][]scraper.Timeslot)
	earliestTimes := make(map[string]int)

	for layout, timeslots := range availableTimes {
		for _, timeSlot := range timeslots {
			gameTimeMinutes, err := parseTimeToMinutes(timeSlot.Time)
			if err != nil {
				continue
			}

			if (filterStartMinutes != 0 || filterEndMinutes != 0) &&
				(gameTimeMinutes < filterStartMinutes || gameTimeMinutes > filterEndMinutes) {
				continue
			}

			layoutTimes[layout] = append(layoutTimes[layout], timeSlot)
			if earliestTime, exists := earliestTimes[layout]; !exists || gameTimeMinutes < earliestTime {
				earliestTimes[layout] = gameTimeMinutes
			}
		}

		sort.Slice(layoutTimes[layout], func(i, j int) bool {
			timeIMinutes, _ := parseTimeToMinutes(layoutTimes[layout][i].Time)
			timeJMinutes, _ := parseTimeToMinutes(layoutTimes[layout][j].Time)
			return timeIMinutes < timeJMinutes
		})
	}

	sortedLayouts := make([]string, 0, len(earliestTimes))
	for layout := range layoutTimes {
		sortedLayouts = append(sortedLayouts, layout)
	}

	sort.Slice(sortedLayouts, func(i, j int) bool {
		return earliestTimes[sortedLayouts[i]] < earliestTimes[sortedLayouts[j]]
	})

	return sortedLayouts, layoutTimes
}

func displaySortedTimes(layoutTimes map[string][]scraper.Timeslot, sortedLayouts []string) {
	fmt.Println("Available times:")
	for _, layout := range sortedLayouts {
		fmt.Printf("\n%s:\n", layout)
		for _, timeSlot := range layoutTimes[layout] {
			fmt.Printf("%s: %d spots available\n", timeSlot.Time, timeSlot.AvailableSpots)
		}
	}
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func isStandardGame(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "9 holes" || n == "18 holes" || n == "twilight"
}

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
