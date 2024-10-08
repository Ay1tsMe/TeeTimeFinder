package cmd

import (
	"TeeTimeFinder/pkg/scraper"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

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
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runScraper() {
	fmt.Println("Starting Golf Scraper...")
	// URLs to scrape
	urls := []string{
		"https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		"https://pointwalter.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		"https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
	}

	// Prompt the user for a date
	fmt.Print("Enter the date (DD-MM): ")
	reader := bufio.NewReader(os.Stdin)
	dateInput, _ := reader.ReadString('\n')
	dateInput = strings.TrimSpace(dateInput)

	// Parse the date input into day and month integers
	day, month, err := parseDayMonth(dateInput)
	if err != nil {
		fmt.Println("Invalid date format. Please use DD-MM.")
		return
	}
	fmt.Printf("Parsed day: %d, month: %d\n", day, month)

	// Build a map of date indices to day and month
	dateIndex := -1
	for i := 0; i <= 4; i++ {
		date := time.Now().AddDate(0, 0, i)
		d := date.Day()
		m := int(date.Month())
		fmt.Printf("Checking date index %d: day %d, month %d\n", i, d, m)
		if d == day && m == month {
			dateIndex = i
			fmt.Printf("Matched date at index %d\n", i)
			break
		}
	}

	if dateIndex == -1 {
		fmt.Println("Selected date is out of range (today to 5 days ahead).")
		return
	}

	// Categorise games
	var standardGames, promoGames []string
	gameToTimeslotURLs := make(map[string][]string)

	// Iterate over the URLs and call the Scrape function
	for _, url := range urls {
		fmt.Printf("Scraping URL: %s\n", url)
		gameTimeslotURLs, err := scraper.Scrape(url, dateIndex)
		if err != nil {
			fmt.Printf("Failed to scrape %s: %v\n", url, err)
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
				if strings.Contains(strings.ToLower(normalisedName), "18 holes") {
					normalisedName = "18 Holes"
				} else if strings.Contains(strings.ToLower(normalisedName), "9 holes") {
					normalisedName = "9 Holes"
				}
			}

			fmt.Printf("Found available game: '%s'\n", normalisedName)

			// Categorise games
			if isStandardGame(normalisedName) {
				standardGames = append(standardGames, normalisedName)
			} else {
				promoGames = append(promoGames, normalisedName)
			}

			// Track the timeslot URLs for each game
			gameToTimeslotURLs[normalisedName] = append(gameToTimeslotURLs[normalisedName], timeslotURL)
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

		// Get the timeslot URLs offering this game
		urlsOfferingGame := gameToTimeslotURLs[selectedGame]
		fmt.Printf("You selected: %s\n", selectedGame)

		fmt.Println("The following timeslot URLs offer this game:")
		for _, timeslotURL := range urlsOfferingGame {
			fmt.Println(timeslotURL) // Print the timeslot URLs
		}

		// Go back to the selection menu after displaying URLs
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
