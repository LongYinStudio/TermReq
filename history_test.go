package main

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadHistoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "termreq", "history.json")
	entries := []HistoryEntry{
		{
			Request: RequestSpec{
				Method:  "POST",
				URL:     "https://api.example.test/widgets",
				Timeout: "10s",
				Headers: []headerPair{{Key: "Content-Type", Value: "application/json"}},
				Body:    `{"hello":"world"}`,
			},
		},
	}

	if err := saveHistory(path, entries); err != nil {
		t.Fatalf("saveHistory returned error: %v", err)
	}

	loaded, err := loadHistory(path)
	if err != nil {
		t.Fatalf("loadHistory returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	if got := loaded[0].Request.Method; got != "POST" {
		t.Fatalf("unexpected method: %q", got)
	}
	if got := loaded[0].Request.URL; got != "https://api.example.test/widgets" {
		t.Fatalf("unexpected URL: %q", got)
	}
	if got := loaded[0].Request.Timeout; got != "10s" {
		t.Fatalf("unexpected timeout: %q", got)
	}
}

func TestAppendHistoryMovesDuplicateToTop(t *testing.T) {
	model := newUIModelWithHistory("", []HistoryEntry{
		{Request: RequestSpec{Method: "GET", URL: "https://example.com/one"}},
		{Request: RequestSpec{Method: "POST", URL: "https://example.com/two"}},
	})

	if err := model.appendHistory(RequestSpec{Method: "POST", URL: "https://example.com/two"}); err != nil {
		t.Fatalf("appendHistory returned error: %v", err)
	}

	if len(model.historyEntries) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(model.historyEntries))
	}
	if got := model.historyEntries[0].Request.URL; got != "https://example.com/two" {
		t.Fatalf("expected duplicate to move to top, got %q", got)
	}
	if got := model.historyEntries[1].Request.URL; got != "https://example.com/one" {
		t.Fatalf("expected previous entry to remain second, got %q", got)
	}
}
