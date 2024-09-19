package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func Scrape(url string, dataDateIndex int) ([]string, error) {
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

	var rowNames []string

	// Cycle through home page and print each game type
	c.OnHTML("div.feeGroupRow", func(e *colly.HTMLElement) {
		// Extract row heading
		rowHeading := e.DOM.Find("div.row-heading > h3").Text()
		fmt.Printf("Row heading: '%s'\n", rowHeading)
		rowHeading = strings.TrimSpace(rowHeading)

		if rowHeading == "" {
			fmt.Println("Row heading is empty, skipping this row.")
			return
		}

		// Find the cell corresponding to the selected data-date index
		cellSelector := fmt.Sprintf("div.items-wrapper > div.cell[data-date='%d']", dataDateIndex)
		fmt.Printf("Looking for cell with selector: '%s'\n", cellSelector)
		cell := e.DOM.Find(cellSelector)

		if cell.Length() == 0 {
			fmt.Printf("No cell found for data-date='%d' in row '%s'\n", dataDateIndex, rowHeading)
			return
		}

		// Check if the cell contains "Not Available"
		cellText := strings.TrimSpace(cell.Text())
		fmt.Printf("Cell text for '%s': '%s'\n", rowHeading, cellText)

		if strings.Contains(strings.ToLower(cellText), "not available") ||
			strings.Contains(strings.ToLower(cellText), "no bookings available") ||
			cellText == "" {
			fmt.Printf("Cell for '%s' is not available or empty.\n", rowHeading)
			return
		}

		// Include the row
		fmt.Printf("Adding '%s' to rowNames\n", rowHeading)
		rowNames = append(rowNames, rowHeading)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}

	c.Wait()
	return rowNames, nil
}
