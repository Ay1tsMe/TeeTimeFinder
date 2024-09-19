package main

import (
	"TeeTimeFinder/pkg/scraper"
	"fmt"
)

func main() {
	fmt.Println("Starting Golf Scraper...")
	// URLs to scrape
	urls := []string{
		"https://fremantlepublic.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?bookingResourceId=3000000&selectedDate=2024-09-19&weekends=false",
		"https://pointwalter.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
		"https://bookings.collierparkgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000",
	}

	// Iterate over the URLs and call the Scrape function
	for _, url := range urls {
		fmt.Printf("Scraping URL: %s\n", url)
		scraper.Scrape(url)
	}
}
