package mathoverflow_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/mathoverflow-cli/mathoverflow"
)

func newTestClient(srv *httptest.Server) *mathoverflow.Client {
	cfg := mathoverflow.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return mathoverflow.NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		type env struct {
			Items   []any `json:"items"`
			HasMore bool  `json:"has_more"`
		}
		_ = json.NewEncoder(w).Encode(env{Items: []any{}})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _ = c.Questions(context.Background(), mathoverflow.QuestionOptions{Limit: 1})
}

func TestQuestionsDecodes(t *testing.T) {
	const payload = `{
		"items": [
			{
				"question_id": 42,
				"title": "What is 6 times 7?",
				"score": 100,
				"view_count": 5000,
				"answer_count": 3,
				"is_answered": true,
				"tags": ["number-theory","analytic"],
				"creation_date": 1609459200,
				"link": "https://mathoverflow.net/questions/42"
			}
		],
		"has_more": false
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	qs, err := c.Questions(context.Background(), mathoverflow.QuestionOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 {
		t.Fatalf("got %d questions, want 1", len(qs))
	}
	q := qs[0]
	if q.ID != 42 {
		t.Errorf("ID = %d, want 42", q.ID)
	}
	if q.Score != 100 {
		t.Errorf("Score = %d, want 100", q.Score)
	}
	if q.Tags != "number-theory,analytic" {
		t.Errorf("Tags = %q", q.Tags)
	}
	if q.URL != "https://mathoverflow.net/questions/42" {
		t.Errorf("URL = %q", q.URL)
	}
}

func TestSearchDecodes(t *testing.T) {
	const payload = `{
		"items": [
			{"question_id": 7, "title": "Riemann hypothesis proof?", "score": 50, "tags": ["nt.number-theory"], "creation_date": 1609459200, "link": "https://mathoverflow.net/questions/7"}
		],
		"has_more": false
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("intitle") == "" {
			t.Error("intitle param missing from search request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	qs, err := c.Search(context.Background(), mathoverflow.SearchOptions{Query: "Riemann", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 {
		t.Fatalf("got %d results, want 1", len(qs))
	}
	if qs[0].ID != 7 {
		t.Errorf("ID = %d, want 7", qs[0].ID)
	}
}

func TestAnswersDecodes(t *testing.T) {
	const payload = `{
		"items": [
			{"answer_id": 99, "question_id": 42, "score": 30, "is_accepted": true, "creation_date": 1609459200}
		],
		"has_more": false
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ans, err := c.Answers(context.Background(), 42, mathoverflow.AnswerOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(ans) != 1 {
		t.Fatalf("got %d answers, want 1", len(ans))
	}
	a := ans[0]
	if a.ID != 99 {
		t.Errorf("ID = %d, want 99", a.ID)
	}
	if !a.IsAccepted {
		t.Error("IsAccepted = false, want true")
	}
}

func TestTagsDecodes(t *testing.T) {
	const payload = `{
		"items": [
			{"name": "nt.number-theory", "count": 3000},
			{"name": "ag.algebraic-geometry", "count": 2500}
		],
		"has_more": false
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	tags, err := c.Tags(context.Background(), mathoverflow.TagOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	if tags[0].Name != "nt.number-theory" {
		t.Errorf("Name = %q", tags[0].Name)
	}
	if tags[0].Count != 3000 {
		t.Errorf("Count = %d, want 3000", tags[0].Count)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"items":[],"has_more":false}`))
	}))
	defer srv.Close()

	cfg := mathoverflow.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := mathoverflow.NewClient(cfg)

	start := time.Now()
	_, err := c.Questions(context.Background(), mathoverflow.QuestionOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}
