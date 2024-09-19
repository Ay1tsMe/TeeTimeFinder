package main

import (
	"TeeTimeFinder/pkg/scraper"
	"flag"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Starting Golf Scraper...")

	url := flag.String("url", "", "URL of the golf booking site")
	flag.Parse()

	if *url == "" {
		log.Fatal("Please provide the URL using -url flag")
	}

	scraper.Scrape(*url)
}
