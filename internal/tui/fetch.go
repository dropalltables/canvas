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

type fetchResult struct {
	desc     string
	locked   bool
	unlockAt *time.Time
}

func fetchDescription(logger *log.Logger, client *api.Client, a api.Assignment) fetchResult {
	assignmentID := a.PlannableID
	if a.PlannableType != "assignment" {
		if a.Plannable.AssignmentID == nil {
			locked := a.Plannable.UnlockAt != nil && time.Now().Before(*a.Plannable.UnlockAt)
			return fetchResult{locked: locked, unlockAt: a.Plannable.UnlockAt}
		}
		assignmentID = *a.Plannable.AssignmentID
	}

	detail, err := client.GetAssignmentDetail(a.CourseID, assignmentID)
	if err != nil {
		logger.Warn("failed to fetch detail", "title", a.Plannable.Title, "error", err)
		locked := a.Plannable.UnlockAt != nil && time.Now().Before(*a.Plannable.UnlockAt)
		return fetchResult{locked: locked, unlockAt: a.Plannable.UnlockAt}
	}

	locked := detail.UnlockAt != nil && time.Now().Before(*detail.UnlockAt)

	if detail.Description == "" {
		return fetchResult{locked: locked, unlockAt: detail.UnlockAt}
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
			return ""
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
		desc += "\n[URL]" + u
	}

	return fetchResult{desc: desc, locked: locked, unlockAt: detail.UnlockAt}
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

	now := time.Now()
	var toFetch []int
	for i, a := range assignments {
		if a.PlannableType == "assignment" {
			toFetch = append(toFetch, i)
		} else if a.Plannable.AssignmentID != nil {
			toFetch = append(toFetch, i)
		} else if a.Plannable.UnlockAt != nil && now.Before(*a.Plannable.UnlockAt) {
			assignments[i].Locked = true
		}
	}
	total := len(toFetch)

	type result struct {
		index    int
		id       int
		desc     string
		locked   bool
		unlockAt *time.Time
		title    string
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
			fr := fetchDescription(logger, client, assignment)
			<-sem
			results <- result{
				index:    idx,
				id:       assignment.PlannableID,
				desc:     fr.desc,
				locked:   fr.locked,
				unlockAt: fr.unlockAt,
				title:    assignment.Plannable.Title,
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
			assignments[r.index].UnlockAt = r.unlockAt
		}
	}

	logger.Info("loading TUI")
	time.Sleep(300 * time.Millisecond)

	p := tea.NewProgram(New(client, assignments, descCache), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
