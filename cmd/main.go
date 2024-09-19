package main

import (
	"TeeTimeFinder/pkg/scraper"
	"fmt"
	"strings"
)

func main() {
	fmt.Println("Starting Golf Scraper...")
	// URLs to scrape
	urls := []string{
		"https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?bookingResourceId=3000000&selectedDate=2024-09-19&weekends=false",
		"https://pointwalter.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		"https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
	}

	// Create a map to store unique row names
	rowNameSet := make(map[string]struct{})

	// Iterate over the URLs and call the Scrape function
	for _, url := range urls {

		fmt.Printf("Scraping URL: %s\n", url)
		rowNames, err := scraper.Scrape(url)

		if err != nil {
			fmt.Printf("Failed to scrape %s: %v\n", url, err)
			continue
		}

		for _, name := range rowNames {
			// Normalize the name to combine like rows (e.g., trim spaces)
			normalisedName := strings.TrimSpace(name)
			rowNameSet[normalisedName] = struct{}{}
		}
	}

	// Collect the unique row names into a slice
	var uniqueRowNames []string
	for name := range rowNameSet {
		uniqueRowNames = append(uniqueRowNames, name)
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
}
