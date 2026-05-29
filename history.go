package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	historyFileVersion = 1
	maxHistoryEntries  = 50
)

type RequestSpec struct {
	Method  string       `json:"method"`
	URL     string       `json:"url"`
	Timeout string       `json:"timeout"`
	Headers []headerPair `json:"headers,omitempty"`
	Body    string       `json:"body,omitempty"`
}

type HistoryEntry struct {
	SavedAt time.Time   `json:"saved_at"`
	Request RequestSpec `json:"request"`
}

type historyFile struct {
	Version int            `json:"version"`
	Entries []HistoryEntry `json:"entries"`
}

func defaultHistoryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "termreq", "history.json"), nil
}

func loadHistory(path string) ([]HistoryEntry, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var payload historyFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	entries := make([]HistoryEntry, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		entry.Request = normalizeRequestSpec(entry.Request)
		if isEmptyRequestSpec(entry.Request) {
			continue
		}
		entries = append(entries, entry)
		if len(entries) >= maxHistoryEntries {
			break
		}
	}
	return entries, nil
}

func saveHistory(path string, entries []HistoryEntry) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	payload := historyFile{
		Version: historyFileVersion,
		Entries: entries,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func normalizeRequestSpec(spec RequestSpec) RequestSpec {
	spec.Method = strings.ToUpper(strings.TrimSpace(spec.Method))
	if spec.Method == "" {
		spec.Method = "GET"
	}
	spec.URL = strings.TrimSpace(spec.URL)
	spec.Timeout = strings.TrimSpace(spec.Timeout)
	spec.Headers = compactHeaderPairs(spec.Headers)
	return spec
}

func compactHeaderPairs(headers []headerPair) []headerPair {
	compacted := make([]headerPair, 0, len(headers))
	for _, header := range headers {
		key := strings.TrimSpace(header.Key)
		value := strings.TrimSpace(header.Value)
		if key == "" && value == "" {
			continue
		}
		compacted = append(compacted, headerPair{Key: key, Value: value})
	}
	return compacted
}

func isEmptyRequestSpec(spec RequestSpec) bool {
	return spec.URL == "" && len(spec.Headers) == 0 && strings.TrimSpace(spec.Body) == "" && strings.TrimSpace(spec.Timeout) == ""
}

func requestSpecsEqual(left, right RequestSpec) bool {
	left = normalizeRequestSpec(left)
	right = normalizeRequestSpec(right)
	if left.Method != right.Method || left.URL != right.URL || left.Timeout != right.Timeout || left.Body != right.Body {
		return false
	}
	if len(left.Headers) != len(right.Headers) {
		return false
	}
	for index := range left.Headers {
		if left.Headers[index] != right.Headers[index] {
			return false
		}
	}
	return true
}
