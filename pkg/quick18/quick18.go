// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

package quick18

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"TeeTimeFinder/pkg/shared"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// Duplicate declarations.
// TODO Remove in the future
var parenthesisRegex = regexp.MustCompile(`\(.+?\)`)
var nineHoleRegex = regexp.MustCompile(`\b9\s*hole(s)?\b`)
var eighteenHoleRegex = regexp.MustCompile(`\b18\s*hole(s)?\b`)
var reSpaceAMPMRegex = regexp.MustCompile(`(\d+:\d+)(AM|PM)\b`)

var allowedStandardModifiers = map[string]bool{
	"walking":  true,
	"midweek":  true,
	"carts":    true, // if "carts can be added" was inside parentheses, it’s already removed, but "carts" alone might remain
	"can":      true,
	"be":       true,
	"added":    true,
	"maylands": true,
}

type Timeslot struct {
	Time           string
	AvailableSpots int
}

func ScrapeDates(baseURL string, selectedDate time.Time) (map[string]string, error) {
	// Example: https://springs.quick18.com/teetimes/searchmatrix?teedate=20250211
	dateStr := selectedDate.Format("20060102") // e.g. "20250211"

	// Parse the base URL to update the "teedate" parameter
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid base URL: %v", err)
	}

	// Overwrite the "teedate" parameter with the chosen date
	q := parsed.Query()
	q.Set("teedate", dateStr)
	parsed.RawQuery = q.Encode()

	finalURL := parsed.String()

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

	// Load the page, look for all <th class="matrixHdrSched">, which are the “game type” columns
	var schedHeaders []string
	c.OnHTML("table.matrixTable thead tr", func(h *colly.HTMLElement) {
		h.ForEach("th.matrixHdrSched", func(_ int, th *colly.HTMLElement) {
			gameName := strings.TrimSpace(th.Text) // e.g. "9 Holes"
			if gameName != "" {
				schedHeaders = append(schedHeaders, gameName)
			}
		})
	})

	var columnHasAvailability []bool

	// Check the row in <tbody>
	c.OnHTML("table.matrixTable tbody tr", func(e *colly.HTMLElement) {
		// Find all .matrixsched cells in this row:
		tdList := e.DOM.Find("td.matrixsched")
		if columnHasAvailability == nil {
			// Initialise the slice once we know how many columns
			columnHasAvailability = make([]bool, tdList.Length())
		}

		// For each column index i, see if it’s active (.mtrxInactive? no) and has a “Select” link
		tdList.Each(func(i int, sel *goquery.Selection) {
			if sel.HasClass("mtrxInactive") {
				return
			}
			selectLinkCount := sel.Find("a.sexybutton.teebutton").Length()
			if selectLinkCount > 0 {
				// This column i has at least one real available time
				columnHasAvailability[i] = true
			}
		})
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("[Quick18] ScrapeDates error:", err)
	})

	if err := c.Visit(finalURL); err != nil {
		return nil, fmt.Errorf("failed to fetch Quick18 date page: %v", err)
	}
	c.Wait()

	gameMap := make(map[string]string)
	for i, header := range schedHeaders {
		if i < len(columnHasAvailability) && columnHasAvailability[i] {
			gameMap[header] = finalURL
		}
	}

	// If no headers were found, fall back to single "All Tee Times"
	if len(gameMap) == 0 {
		gameMap["All Tee Times"] = finalURL
	}

	return gameMap, nil
}

// ScrapeTimes visits the Quick18 "matrixTable" page and extracts timeslots
func ScrapeTimes(url string) (map[string][]shared.TeeTimeSlot, error) {
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(1),
	)

	// Rate limiting, etc.
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	// 1) Grab all the column headers (e.g. "9 Holes", "18 Holes", etc.)
	var columnHeaders []string
	c.OnHTML("table.matrixTable thead tr", func(h *colly.HTMLElement) {
		h.ForEach("th.matrixHdrSched", func(_ int, th *colly.HTMLElement) {
			headerText := strings.TrimSpace(th.Text)
			if headerText != "" {
				normalise := normaliseGameName(headerText)
				columnHeaders = append(columnHeaders, normalise)
			}
		})
	})

	// 2) Prepare a map from “header text” -> slice of tee times.
	headerToTimes := make(map[string][]shared.TeeTimeSlot)

	// 3) For each body row, parse the time, players, and each sched cell.
	c.OnHTML("table.matrixTable tbody tr", func(e *colly.HTMLElement) {
		// Time cell
		rawTime := strings.TrimSpace(e.ChildText("td.mtrxTeeTimes"))
		timeStr := parseTimeCell(rawTime)

		// Players cell
		playerCell := strings.TrimSpace(e.ChildText("td.matrixPlayers"))
		availableSpots := parsePlayers(playerCell)

		// The “sched” cells (one per header)
		schedCells := e.DOM.Find("td.matrixsched")
		schedCells.Each(func(i int, sel *goquery.Selection) {
			// Make sure we don’t run past columnHeaders
			if i >= len(columnHeaders) {
				return
			}
			header := columnHeaders[i] // e.g. "9 Holes", "18 Holes", etc.

			// Skip if it’s inactive
			if sel.HasClass("mtrxInactive") {
				return
			}
			// Does this cell have a "Select" link?
			linkCount := sel.Find("a.sexybutton.teebutton").Length()
			if linkCount == 0 {
				return
			}

			// If we reach here, it's an available slot for that header.
			slot := shared.TeeTimeSlot{
				Time:           timeStr,
				AvailableSpots: availableSpots,
			}
			headerToTimes[header] = append(headerToTimes[header], slot)
		})
	})

	// Handle errors and visit
	c.OnError(func(_ *colly.Response, err error) {
		log.Println("[Quick18] Error:", err)
	})
	if err := c.Visit(url); err != nil {
		return nil, fmt.Errorf("failed to visit Quick18 URL %s: %v", url, err)
	}
	c.Wait()

	return headerToTimes, nil
}

// parseTimeCell merges something like "2:30\nPM" into "2:30 PM"
func parseTimeCell(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 2 {
		return strings.TrimSpace(lines[0]) + " " + strings.TrimSpace(lines[1])
	}
	return strings.ReplaceAll(raw, "\n", " ")
}

// parsePlayers tries to extract the maximum # of players from something like "1 to 4 players" or "1 player"
func parsePlayers(cellText string) int {
	s := strings.ToLower(cellText)
	s = strings.ReplaceAll(s, "players", "") // e.g. "1 to 4 "
	s = strings.ReplaceAll(s, "player", "")  // e.g. "1 " or "1 to 4 "

	// We'll look for the last digit. For example:
	//  "1 to 4 " => should capture "4"
	re := regexp.MustCompile(`(\d+)\s*$`)
	match := re.FindStringSubmatch(s)
	if len(match) == 2 {
		maxP, err := strconv.Atoi(match[1])
		if err == nil && maxP > 0 {
			return maxP
		}
	}

	// If no match, default to 1
	return 1
}

// Helper for normalising game types
// Duplicate code! Need to get rid of it and seperate helpers from root.go
func normaliseGameName(originalName string) string {
	// Lowercase and trim
	name := strings.ToLower(strings.TrimSpace(originalName))
	// Remove parentheses and their content
	name = parenthesisRegex.ReplaceAllString(name, "")
	name = strings.TrimSpace(name)

	// Check if it contains 9 or 18 hole references
	hasNine := nineHoleRegex.MatchString(name)
	hasEighteen := eighteenHoleRegex.MatchString(name)

	// If no hole count found, it's a promo
	if !hasNine && !hasEighteen {
		return strings.Title(name)
	}

	// Use regex to safely replace "9 hole(s)" with "9 holes"
	name = nineHoleRegex.ReplaceAllString(name, "9 holes")
	// Use regex to safely replace "18 hole(s)" with "18 holes"
	name = eighteenHoleRegex.ReplaceAllString(name, "18 holes")

	// Split into words
	words := strings.Fields(name)

	// Remove the "9 holes" or "18 holes" from words
	filtered := []string{}
	skipNext := false
	for i, w := range words {
		if w == "9" && i+1 < len(words) && words[i+1] == "holes" {
			skipNext = true
			continue
		}
		if w == "18" && i+1 < len(words) && words[i+1] == "holes" {
			skipNext = true
			continue
		}
		if skipNext {
			skipNext = false
			continue
		}
		filtered = append(filtered, w)
	}

	// Remove allowed standard modifiers
	finalWords := []string{}
	for _, w := range filtered {
		if !allowedStandardModifiers[w] {
			finalWords = append(finalWords, w)
		}
	}

	// If no extra words remain, it's a pure standard game
	if len(finalWords) == 0 {
		if hasNine {
			return "9 Holes"
		}
		if hasEighteen {
			return "18 Holes"
		}
	}

	// Otherwise, it's a promo
	return strings.Title(name)
}
