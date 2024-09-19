package main

import (
	"TeeTimeFinder/pkg/scraper"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("Starting Golf Scraper...")
	// URLs to scrape
	urls := []string{
		"https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?bookingResourceId=3000000&selectedDate=2024-09-19&weekends=false",
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

	// Create a map to store unique row names
	rowNameSet := make(map[string]struct{})
	rowNameToURLs := make(map[string][]string)

	// Iterate over the URLs and call the Scrape function
	for _, url := range urls {

		fmt.Printf("Scraping URL: %s\n", url)
		rowNames, err := scraper.Scrape(url, dateIndex)

		if err != nil {
			fmt.Printf("Failed to scrape %s: %v\n", url, err)
			continue
		}

		for _, name := range rowNames {
			// Normalise the name to combine like rows (e.g., trim spaces)
			normalisedName := strings.TrimSpace(name)
			fmt.Printf("Found available game: '%s'\n", normalisedName)
			rowNameSet[normalisedName] = struct{}{}
			rowNameToURLs[normalisedName] = append(rowNameToURLs[normalisedName], url)
		}
	}

	// Collect the unique row names into a slice
	var uniqueRowNames []string
	for name := range rowNameSet {
		fmt.Printf("Adding '%s' to unique row names\n", name)
		uniqueRowNames = append(uniqueRowNames, name)
	}

	if len(uniqueRowNames) == 0 {
		fmt.Println("No available games found on the selected date.")
		return
	}

	// Now, present the options to the user
	fmt.Println("Select what game you want to play:")
	for i, name := range uniqueRowNames {
		fmt.Printf("%d. %s\n", i+1, name)
	}

	// Read user input
	var choice int
	fmt.Print("Enter the number of your choice: ")
	fmt.Scanln(&choice)

	if choice < 1 || choice > len(uniqueRowNames) {
		fmt.Println("Invalid choice")
		return
	}

	selectedGame := uniqueRowNames[choice-1]
	fmt.Printf("You selected: %s\n", selectedGame)

	// Get the URLs offering this game
	urlsOfferingGame := rowNameToURLs[selectedGame]
	fmt.Println("The following courses offer this game:")
	for _, url := range urlsOfferingGame {
		fmt.Println(url)
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
