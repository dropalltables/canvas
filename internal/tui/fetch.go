package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/dropalltables/canvas/internal/api"
)

func fetchDescription(logger *log.Logger, client *api.Client, a api.Assignment) (string, bool) {
	if a.PlannableType != "assignment" {
		return "", false
	}

	detail, err := client.GetAssignmentDetail(a.CourseID, a.PlannableID)
	if err != nil {
		logger.Warn("failed to fetch detail", "title", a.Plannable.Title, "error", err)
		return "", false
	}

	locked := detail.UnlockAt != nil && time.Now().Before(*detail.UnlockAt)

	if detail.Description == "" {
		return "", locked
	}

	md, _ := htmltomd.ConvertString(detail.Description)
	if md == "" {
		md = detail.Description
	}

	var urls []string
	text := linkRegex.ReplaceAllStringFunc(md, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			urls = append(urls, parts[2])
			return parts[1]
		}
		return match
	})

	var words []string
	for _, line := range strings.Split(text, "\n") {
		if w := strings.TrimSpace(line); w != "" {
			words = append(words, w)
		}
	}
	desc := strings.Join(words, " ")

	for _, u := range urls {
		desc += "\n- " + u
	}

	return desc, locked
}

func Run(client *api.Client) error {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})

	logger.Info("starting canvas")
	logger.Info("fetching assignments", "past", 14, "future", 30)

	assignments, err := client.GetAssignments(14, 30)
	if err != nil {
		logger.Error("failed to fetch assignments", "error", err)
		return err
	}
	logger.Info("fetched assignments", "count", len(assignments))

	var toFetch []int
	for i, a := range assignments {
		if a.PlannableType == "assignment" {
			toFetch = append(toFetch, i)
		}
	}
	total := len(toFetch)

	type result struct {
		index  int
		id     int
		desc   string
		locked bool
		title  string
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(chan result, total)
	sem := make(chan struct{}, 10)
	fetchNum := 0

	for _, i := range toFetch {
		a := assignments[i]
		wg.Add(1)
		go func(idx int, assignment api.Assignment) {
			defer wg.Done()
			sem <- struct{}{}
			mu.Lock()
			fetchNum++
			num := fetchNum
			mu.Unlock()
			logger.Info(fmt.Sprintf("fetching (%d/%d)", num, total), "title", assignment.Plannable.Title)
			desc, locked := fetchDescription(logger, client, assignment)
			<-sem
			results <- result{
				index:  idx,
				id:     assignment.PlannableID,
				desc:   desc,
				locked: locked,
				title:  assignment.Plannable.Title,
			}
		}(i, a)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	descCache := make(map[int]string)
	fetched := 0
	for r := range results {
		fetched++
		logger.Info(fmt.Sprintf("fetched (%d/%d)", fetched, total), "title", r.title, "locked", r.locked)
		if r.desc != "" {
			descCache[r.id] = r.desc
		}
		if r.locked {
			assignments[r.index].Locked = true
		}
	}

	logger.Info("loading TUI")
	time.Sleep(300 * time.Millisecond)

	p := tea.NewProgram(New(client, assignments, descCache), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
