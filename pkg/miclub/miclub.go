// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

package miclub

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/Ay1tsMe/TeeTimeFinder/pkg/shared"

	"github.com/gocolly/colly"
)

// Scrapes the date URL and returns a map of games and their corresponding timeslot URLs
func ScrapeDates(baseURL string, selectedDate time.Time) (map[string]string, error) {
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

	dateStr := selectedDate.Format("2006-01-02")

	// Parse the base URL
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}

	// Update query parameters
	q := parsedBaseURL.Query()
	q.Set("selectedDate", dateStr)
	q.Set("weekends", "false")
	parsedBaseURL.RawQuery = q.Encode()

	// Visit the URL
	err = c.Visit(parsedBaseURL.String())
	if err != nil {
		return nil, err
	}

	// Keep a copy of the parsed base URL for constructing timeslot URLs
	baseURLCopy := *parsedBaseURL

	// Cycle through the feeGroupRow to capture each game's type and available timeslot
	c.OnHTML("div.feeGroupRow", func(e *colly.HTMLElement) {
		// Extract the row heading (game type)
		rowHeading := e.DOM.Find("div.row-heading > h3").Text()
		rowHeading = strings.TrimSpace(rowHeading)

		if rowHeading == "" {
			return
		}

		// Find the cell corresponding to the selected date (data-date="0")
		cell := e.DOM.Find("div.items-wrapper > div.cell[data-date='0']")
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
			timeslotURL := constructTimeslotURL(&baseURLCopy, onclickAttr)
			if timeslotURL != "" {
				// Store the row heading and its corresponding timeslot URL
				rowNameToTimeslotURL[rowHeading] = timeslotURL
			}
		}
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	c.Wait()
	return rowNameToTimeslotURL, nil
}

func ScrapeTimes(url string) (map[string][]shared.TeeTimeSlot, error) {
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
	layoutToTimes := make(map[string][]shared.TeeTimeSlot)

	c.OnHTML("div.row-time", func(e *colly.HTMLElement) {

		// Extract the time
		time := e.ChildText("div.time-wrapper > h3")
		time = strings.TrimSpace(time)

		// Extract the layout (course configuration)
		layout := e.ChildText("div.time-wrapper > h4")
		layout = strings.TrimSpace(layout)

		if layout == "" || time == "" {
			return
		}

		availableSlots := e.DOM.Find("div.cell.cell-available").Length()

		// Only include times with available slots
		if availableSlots > 0 {
			timeSlot := shared.TeeTimeSlot{
				Time:           time,
				AvailableSpots: availableSlots,
			}

			// Add this timeSlot to the layout
			layoutToTimes[layout] = append(layoutToTimes[layout], timeSlot)
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
	return layoutToTimes, nil

}

// Helper function to construct the full timeslot URL based on the onclick attribute
func constructTimeslotURL(parsedBaseURL *url.URL, onclickAttr string) string {
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

			// Copy the base URL to avoid modifying the original
			timeslotURL := *parsedBaseURL

			// Update path to /guests/bookings/ViewPublicTimesheet.msp
			timeslotURL.Path = "/guests/bookings/ViewPublicTimesheet.msp"

			// Update query parameters
			q := timeslotURL.Query()
			q.Set("feeGroupId", feeGroupID)
			q.Set("selectedDate", selectedDate)
			q.Set("weekends", "false")
			timeslotURL.RawQuery = q.Encode()

			return timeslotURL.String()
		}
	}
	return ""
}
