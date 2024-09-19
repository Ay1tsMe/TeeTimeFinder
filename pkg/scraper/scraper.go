package scraper

import (
	"fmt"
	"log"

	"github.com/gocolly/colly"
)

func Scrape(url string) {
	c := colly.NewCollector()

	c.OnHTML("div.available-slot", func(e *colly.HTMLElement) {
		timeSlot := e.Text
		fmt.Println("Available Slot:", timeSlot)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	c.Visit(url)
}
