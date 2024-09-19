package scraper

import (
	"fmt"
	"log"

	"github.com/gocolly/colly"
)

func Scrape(url string) {
	c := colly.NewCollector()

	// Cycle through home page and print each game type
	c.OnHTML("div.feeGroupRow", func(e *colly.HTMLElement) {
		// Extract row heading
		rowHeading := e.DOM.Find("div.row-heading > h3").Text()
		fmt.Println(rowHeading)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	c.Visit(url)
}
