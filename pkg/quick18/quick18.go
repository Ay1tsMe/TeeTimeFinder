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

	"github.com/gocolly/colly"
)

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

	gameMap := make(map[string]string)

	// Load the page, look for all <th class="matrixHdrSched">, which are the “game type” columns
	c.OnHTML("table.matrixTable thead tr", func(h *colly.HTMLElement) {
		h.ForEach("th.matrixHdrSched", func(_ int, th *colly.HTMLElement) {
			gameName := strings.TrimSpace(th.Text) // e.g. "9 Holes"
			if gameName == "" {
				return
			}
			// Store each header as a separate “game” => same finalURL
			// Because Quick18 does not provide a separate URL per column
			gameMap[gameName] = finalURL
		})
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("[Quick18] ScrapeDates error:", err)
	})

	if err := c.Visit(finalURL); err != nil {
		return nil, fmt.Errorf("failed to fetch Quick18 date page: %v", err)
	}
	c.Wait()

	// If no headers were found, fall back to single "All Tee Times"
	if len(gameMap) == 0 {
		gameMap["All Tee Times"] = finalURL
	}

	return gameMap, nil
}

// ScrapeTimes visits the Quick18 "matrixTable" page and extracts timeslots
// grouped by "layout" (the course column).
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

	layoutToTimes := make(map[string][]shared.TeeTimeSlot)

	c.OnHTML("table.matrixTable tbody tr", func(e *colly.HTMLElement) {
		// Extract time from the "td.mtrxTeeTimes" cell.
		rawTime := strings.TrimSpace(e.ChildText("td.mtrxTeeTimes"))
		// Typically something like "2:30\nPM" -> we can normalize:
		timeStr := parseTimeCell(rawTime)

		// Extract layout (e.g. "9 Holes" or "18 Holes (Double Loop)")
		layout := strings.TrimSpace(e.ChildText("td.mtrxCourse"))
		if layout == "" {
			// skip row
			return
		}

		// Extract the "matrixPlayers" cell for player range
		playerCell := strings.TrimSpace(e.ChildText("td.matrixPlayers"))
		availableSpots := parsePlayers(playerCell)

		// Check if there's at least one active "select" link in a .matrixsched cell
		// that is not .mtrxInactive:
		selectLinks := e.ChildAttrs("td.matrixsched:not(.mtrxInactive) a.sexybutton.teebutton", "href")
		if len(selectLinks) == 0 {
			// No active "Select" => row is unavailable
			return
		}

		// We consider it an available timeslot.
		timeslot := shared.TeeTimeSlot{
			Time:           timeStr,
			AvailableSpots: availableSpots,
		}
		layoutToTimes[layout] = append(layoutToTimes[layout], timeslot)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("[Quick18] Error:", err)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit Quick18 URL %s: %v", url, err)
	}

	c.Wait()
	return layoutToTimes, nil
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
