// Package mathoverflow is the library behind the mo command: the HTTP client,
// request shaping, and the typed data models for MathOverflow.
//
// All data is fetched from the public Stack Exchange API v2.3 at
// https://api.stackexchange.com/2.3 with &site=mathoverflow. No API key is
// required for read-only access at the default rate limit.
package mathoverflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the Stack Exchange API.
const DefaultUserAgent = "mo/dev (+https://github.com/tamnd/mathoverflow-cli)"

// baseURL is the Stack Exchange API v2.3 base.
const defaultBaseURL = "https://api.stackexchange.com/2.3"

// Config holds constructor parameters for Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   defaultBaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   5,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Stack Exchange API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    cfg.BaseURL,
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── wire envelope ────────────────────────────────────────────────────────────

type envelope[T any] struct {
	Items   []T  `json:"items"`
	HasMore bool `json:"has_more"`
}

// ─── Questions ───────────────────────────────────────────────────────────────

// QuestionOptions controls the /questions endpoint.
type QuestionOptions struct {
	Sort  string // votes | activity | newest
	Tag   string
	Limit int
}

// Questions fetches a list of questions.
func (c *Client) Questions(ctx context.Context, opts QuestionOptions) ([]Question, error) {
	params := url.Values{}
	params.Set("site", "mathoverflow")
	params.Set("order", "desc")
	sort := opts.Sort
	if sort == "" {
		sort = "votes"
	}
	// Stack Exchange API uses "creation" for newest
	if sort == "newest" {
		sort = "creation"
	}
	params.Set("sort", sort)
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	params.Set("pagesize", strconv.Itoa(limit))
	if opts.Tag != "" {
		params.Set("tagged", opts.Tag)
	}
	rawURL := c.baseURL + "/questions?" + params.Encode()

	var env envelope[wireQuestion]
	if err := c.getJSON(ctx, rawURL, &env); err != nil {
		return nil, err
	}
	out := make([]Question, 0, len(env.Items))
	for _, wq := range env.Items {
		out = append(out, wireToQuestion(wq))
	}
	return out, nil
}

// Question fetches a single question by id.
func (c *Client) Question(ctx context.Context, id int) (Question, error) {
	params := url.Values{}
	params.Set("site", "mathoverflow")
	rawURL := fmt.Sprintf("%s/questions/%d?%s", c.baseURL, id, params.Encode())

	var env envelope[wireQuestion]
	if err := c.getJSON(ctx, rawURL, &env); err != nil {
		return Question{}, err
	}
	if len(env.Items) == 0 {
		return Question{}, fmt.Errorf("question %d not found", id)
	}
	return wireToQuestion(env.Items[0]), nil
}

// ─── Search ──────────────────────────────────────────────────────────────────

// SearchOptions controls the /search endpoint.
type SearchOptions struct {
	Query string
	Sort  string // votes | activity | newest
	Limit int
}

// Search searches questions by title keyword.
func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]Question, error) {
	params := url.Values{}
	params.Set("site", "mathoverflow")
	params.Set("intitle", opts.Query)
	params.Set("order", "desc")
	sort := opts.Sort
	if sort == "" {
		sort = "votes"
	}
	if sort == "newest" {
		sort = "creation"
	}
	params.Set("sort", sort)
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	params.Set("pagesize", strconv.Itoa(limit))

	rawURL := c.baseURL + "/search?" + params.Encode()

	var env envelope[wireQuestion]
	if err := c.getJSON(ctx, rawURL, &env); err != nil {
		return nil, err
	}
	out := make([]Question, 0, len(env.Items))
	for _, wq := range env.Items {
		out = append(out, wireToQuestion(wq))
	}
	return out, nil
}

// ─── Answers ─────────────────────────────────────────────────────────────────

// AnswerOptions controls the /questions/{id}/answers endpoint.
type AnswerOptions struct {
	Limit int
}

// Answers fetches answers for a question.
func (c *Client) Answers(ctx context.Context, questionID int, opts AnswerOptions) ([]Answer, error) {
	params := url.Values{}
	params.Set("site", "mathoverflow")
	params.Set("order", "desc")
	params.Set("sort", "votes")
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	params.Set("pagesize", strconv.Itoa(limit))
	rawURL := fmt.Sprintf("%s/questions/%d/answers?%s", c.baseURL, questionID, params.Encode())

	var env envelope[wireAnswer]
	if err := c.getJSON(ctx, rawURL, &env); err != nil {
		return nil, err
	}
	out := make([]Answer, 0, len(env.Items))
	for _, wa := range env.Items {
		out = append(out, wireToAnswer(wa, questionID))
	}
	return out, nil
}

// ─── Tags ─────────────────────────────────────────────────────────────────────

// TagOptions controls the /tags endpoint.
type TagOptions struct {
	Limit  int
	Search string
}

// Tags fetches the most popular tags.
func (c *Client) Tags(ctx context.Context, opts TagOptions) ([]Tag, error) {
	params := url.Values{}
	params.Set("site", "mathoverflow")
	params.Set("order", "desc")
	params.Set("sort", "popular")
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	params.Set("pagesize", strconv.Itoa(limit))
	if opts.Search != "" {
		params.Set("inname", opts.Search)
	}
	rawURL := c.baseURL + "/tags?" + params.Encode()

	var env envelope[wireTag]
	if err := c.getJSON(ctx, rawURL, &env); err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(env.Items))
	for _, wt := range env.Items {
		out = append(out, Tag{Name: wt.Name, Count: wt.Count})
	}
	return out, nil
}

// ─── wire types ──────────────────────────────────────────────────────────────

type wireQuestion struct {
	QuestionID  int      `json:"question_id"`
	Title       string   `json:"title"`
	Score       int      `json:"score"`
	ViewCount   int      `json:"view_count"`
	AnswerCount int      `json:"answer_count"`
	IsAnswered  bool     `json:"is_answered"`
	Tags        []string `json:"tags"`
	CreationDate int64   `json:"creation_date"`
	Link        string   `json:"link"`
}

type wireAnswer struct {
	AnswerID     int   `json:"answer_id"`
	QuestionID   int   `json:"question_id"`
	Score        int   `json:"score"`
	IsAccepted   bool  `json:"is_accepted"`
	CreationDate int64 `json:"creation_date"`
}

type wireTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func wireToQuestion(wq wireQuestion) Question {
	link := wq.Link
	if link == "" {
		link = fmt.Sprintf("https://mathoverflow.net/questions/%d", wq.QuestionID)
	}
	return Question{
		ID:          wq.QuestionID,
		Title:       stripTags(wq.Title),
		Score:       wq.Score,
		ViewCount:   wq.ViewCount,
		AnswerCount: wq.AnswerCount,
		IsAnswered:  wq.IsAnswered,
		Tags:        strings.Join(wq.Tags, ","),
		CreatedAt:   isoDate(wq.CreationDate),
		URL:         link,
	}
}

func wireToAnswer(wa wireAnswer, questionID int) Answer {
	qid := wa.QuestionID
	if qid == 0 {
		qid = questionID
	}
	return Answer{
		ID:         wa.AnswerID,
		Score:      wa.Score,
		IsAccepted: wa.IsAccepted,
		CreatedAt:  isoDate(wa.CreationDate),
		URL:        fmt.Sprintf("https://mathoverflow.net/a/%d", wa.AnswerID),
	}
}

func isoDate(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", `"`)
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&apos;", "'")
	return strings.TrimSpace(out)
}
