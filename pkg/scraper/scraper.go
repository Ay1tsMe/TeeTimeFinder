package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func Scrape(url string) ([]string, error) {
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
		fmt.Println(rowHeading)
		rowHeading = strings.TrimSpace(rowHeading)
		if rowHeading != "" {
			rowNames = append(rowNames, rowHeading)
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
	return rowNames, nil
}
