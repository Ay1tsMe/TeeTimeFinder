package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

// Scrapes the date URL and returns a map of games and their corresponding timeslot URLs
func ScrapeDates(url string, dataDateIndex int) (map[string]string, error) {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
	)

	// Implement rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	// Map to store the row names and their associated timeslot URLs
	rowNameToTimeslotURL := make(map[string]string)

	// Cycle through the feeGroupRow to capture each game's type and available timeslot
	c.OnHTML("div.feeGroupRow", func(e *colly.HTMLElement) {
		// Extract the row heading (game type)
		rowHeading := e.DOM.Find("div.row-heading > h3").Text()
		rowHeading = strings.TrimSpace(rowHeading)

		if rowHeading == "" {
			return
		}

		// Find the cell corresponding to the selected data-date index
		cellSelector := fmt.Sprintf("div.items-wrapper > div.cell[data-date='%d']", dataDateIndex)
		cell := e.DOM.Find(cellSelector)

		if cell.Length() == 0 {
			return
		}

		// Check if the cell is available (i.e., does not contain "Not Available")
		cellText := strings.TrimSpace(cell.Text())

		if strings.Contains(strings.ToLower(cellText), "not available") ||
			strings.Contains(strings.ToLower(cellText), "no bookings available") ||
			cellText == "" {
			return
		}

		// Extract the "onclick" attribute for the timeslot URL construction
		onclickAttr, exists := cell.Attr("onclick")
		if exists && strings.Contains(onclickAttr, "redirectToTimesheet") {
			// Extract the feeGroupId and selectedDate from the JavaScript function call
			timeslotURL := constructTimeslotURL(url, onclickAttr)
			if timeslotURL != "" {
				// Store the row heading and its corresponding timeslot URL
				rowNameToTimeslotURL[rowHeading] = timeslotURL
			}
		}
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}

	c.Wait()
	return rowNameToTimeslotURL, nil
}

func ScrapeTimes(url string) (map[string]int, error) {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
	)

	// Implement rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	// Stores the available times
	availableTimes := make(map[string]int)

	c.OnHTML("div.row-time", func(e *colly.HTMLElement) {

		// Extract the time
		time := e.ChildText("div.time-wrapper > h3")
		time = strings.TrimSpace(time)

		availableSlots := e.DOM.Find("div.cell.cell-available").Length()

		// Only include times with available slots
		if availableSlots > 0 {
			availableTimes[time] = availableSlots
		}
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error: ", err)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}

	c.Wait()
	return availableTimes, nil

}

// Helper function to construct the full timeslot URL based on the onclick attribute
func constructTimeslotURL(baseURL string, onclickAttr string) string {
	// Example of onclick content: "javascript:redirectToTimesheet('101527','2024-09-22');"
	// We need to extract the feeGroupId ('101527') and the date ('2024-09-22')
	// Then construct the timeslot URL: https://{base_url}/ViewPublicTimesheet.msp?bookingResourceId=3000000&selectedDate={date}&feeGroupId={feeGroupId}

	// Extract the portion between the parentheses
	start := strings.Index(onclickAttr, "(")
	end := strings.Index(onclickAttr, ")")
	if start != -1 && end != -1 && end > start {
		// Extract the parameters
		params := onclickAttr[start+1 : end]
		paramList := strings.Split(params, ",")
		if len(paramList) == 2 {
			feeGroupID := strings.Trim(paramList[0], "' ")
			selectedDate := strings.Trim(paramList[1], "' ")

			// Construct the correct timeslot URL using "/ViewPublicTimesheet.msp"
			timeslotURL := fmt.Sprintf("%s/ViewPublicTimesheet.msp?bookingResourceId=3000000&selectedDate=%s&feeGroupId=%s",
				strings.Split(baseURL, "/ViewPublicCalendar.msp")[0], // Use the base part of the URL before "/ViewPublicCalendar.msp"
				selectedDate,
				feeGroupID)
			return timeslotURL
		}
	}
	return ""
}
