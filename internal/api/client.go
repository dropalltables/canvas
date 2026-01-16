package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dropalltables/canvas/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

type Assignment struct {
	ID            int          `json:"id"`
	PlannableID   int          `json:"plannable_id"`
	PlannableType string       `json:"plannable_type"`
	Plannable     Plannable    `json:"plannable"`
	ContextName   string       `json:"context_name"`
	ContextType   string       `json:"context_type"`
	CourseID      int          `json:"course_id"`
	DueAt         *time.Time   `json:"plannable_date"`
	HTMLURL       string       `json:"html_url"`
	Completed     bool         `json:"-"`
	Submitted     bool         `json:"-"`
	Locked        bool         `json:"-"`
	Override      *Override    `json:"planner_override"`
	Submissions   *Submissions `json:"submissions"`
}

type Submissions struct {
	Submitted bool `json:"submitted"`
	Graded    bool `json:"graded"`
}

func (s *Submissions) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if string(data) == "false" {
		s.Submitted = false
		return nil
	}
	if string(data) == "true" {
		s.Submitted = true
		return nil
	}
	type submissions Submissions
	return json.Unmarshal(data, (*submissions)(s))
}

type Plannable struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	DueAt     *time.Time `json:"due_at"`
	LockAt    *time.Time `json:"lock_at"`
	UnlockAt  *time.Time `json:"unlock_at"`
	Published *bool      `json:"published"`
}

type Override struct {
	ID            int  `json:"id"`
	PlannableID   int  `json:"plannable_id"`
	PlannableType string `json:"plannable_type"`
	MarkedComplete bool `json:"marked_complete"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(cfg.BaseURL, "/") + "/api/v1",
		token:      cfg.Token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		msg := string(bodyBytes)
		if readErr != nil {
			msg = fmt.Sprintf("%s (error reading body: %v)", msg, readErr)
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}

	return resp, nil
}

func plannerItemsPath(now time.Time, pastDays, futureDays int) string {
	start := now.AddDate(0, 0, -pastDays).Format("2006-01-02")
	end := now.AddDate(0, 0, futureDays).Format("2006-01-02")
	return fmt.Sprintf("/planner/items?start_date=%s&end_date=%s&per_page=100", start, end)
}

func (c *Client) GetAssignments(pastDays, futureDays int) ([]Assignment, error) {
	now := time.Now()
	path := plannerItemsPath(now, pastDays, futureDays)
	resp, err := c.do("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items []Assignment
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	var filtered []Assignment
	for _, item := range items {
		if !isValidAssignment(item, now) {
			continue
		}
		if item.Override != nil {
			item.Completed = item.Override.MarkedComplete
		}
		if item.Submissions != nil {
			item.Submitted = item.Submissions.Submitted
		}
		if item.Submitted {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

func isValidAssignment(a Assignment, now time.Time) bool {
	switch a.PlannableType {
	case "assignment", "quiz", "discussion_topic":
	default:
		return false
	}

	if a.DueAt == nil {
		return false
	}

	if a.Plannable.LockAt != nil && now.After(*a.Plannable.LockAt) {
		return false
	}

	if a.Plannable.Published != nil && !*a.Plannable.Published {
		return false
	}

	return true
}

func (c *Client) MarkDone(plannableType string, plannableID int, overrideID *int) error {
	data := url.Values{}
	data.Set("marked_complete", "true")

	var path string
	var method string

	if overrideID != nil {
		path = fmt.Sprintf("/planner/overrides/%d", *overrideID)
		method = "PUT"
	} else {
		path = "/planner/overrides"
		method = "POST"
		data.Set("plannable_type", plannableType)
		data.Set("plannable_id", fmt.Sprintf("%d", plannableID))
	}

	resp, err := c.do(method, path, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) BaseURL() string {
	return strings.TrimSuffix(c.baseURL, "/api/v1")
}

type AssignmentDetail struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CourseID    int        `json:"course_id"`
	UnlockAt    *time.Time `json:"unlock_at"`
}

func (c *Client) GetAssignmentDetail(courseID, assignmentID int) (*AssignmentDetail, error) {
	path := fmt.Sprintf("/courses/%d/assignments/%d", courseID, assignmentID)
	resp, err := c.do("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var detail AssignmentDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func (c *Client) GetRawPlannerItems(pastDays, futureDays int) ([]byte, error) {
	path := plannerItemsPath(time.Now(), pastDays, futureDays)
	resp, err := c.do("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
