package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNormalizeURLAddsHTTPS(t *testing.T) {
	got, err := normalizeURL("example.com/api")
	if err != nil {
		t.Fatalf("normalizeURL returned error: %v", err)
	}
	if got != "https://example.com/api" {
		t.Fatalf("unexpected normalized url: %s", got)
	}
}

func TestParseTimeoutValue(t *testing.T) {
	duration, err := parseTimeoutValue("2.5")
	if err != nil {
		t.Fatalf("parseTimeoutValue returned error: %v", err)
	}
	if duration != 2500*time.Millisecond {
		t.Fatalf("unexpected duration: %s", duration)
	}
}

func TestParseHeaderLines(t *testing.T) {
	headers, err := parseHeaderLines([]string{"Content-Type: application/json", "X-Test: abc"})
	if err != nil {
		t.Fatalf("parseHeaderLines returned error: %v", err)
	}
	if headers.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %q", headers.Get("Content-Type"))
	}
	if headers.Get("X-Test") != "abc" {
		t.Fatalf("unexpected x-test: %q", headers.Get("X-Test"))
	}
}

func TestPerformRequest(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", request.Method)
			}
			if request.Header.Get("X-Trace") != "abc" {
				t.Fatalf("expected X-Trace header to be forwarded")
			}
			if request.URL.String() != "https://example.com/demo" {
				t.Fatalf("unexpected URL: %s", request.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Status:     "201 Created",
				Proto:      "HTTP/1.1",
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Server":     []string{"test"},
				},
				Body: io.NopCloser(strings.NewReader(`{"ok":true,"path":"` + request.URL.Path + `"}`)),
			}, nil
		}),
		Timeout: 2 * time.Second,
	}

	result := performRequestWithClient(client, http.MethodPost, "example.com/demo", []string{"X-Trace: abc"}, `{"hello":"world"}`)
	if result.level != "success" {
		t.Fatalf("expected success level, got %s", result.level)
	}
	if result.view.Status != "201 Created" {
		t.Fatalf("unexpected status: %s", result.view.Status)
	}
	if result.view.Proto == "" {
		t.Fatalf("expected protocol to be populated")
	}
	if !containsLine(result.view.HeaderLines, "X-Server: test") {
		t.Fatalf("expected response headers to contain X-Server")
	}
	if !containsSubstring(result.view.BodyLines, `"ok": true`) {
		t.Fatalf("expected pretty JSON body, got %v", result.view.BodyLines)
	}
}

func TestFormatResponseBodyBinary(t *testing.T) {
	lines := formatResponseBody([]byte{0x00, 0x01, 0x02}, "application/octet-stream")
	if len(lines) == 0 || !strings.Contains(lines[0], "binary payload") {
		t.Fatalf("expected binary payload note, got %v", lines)
	}
}

func containsLine(lines []string, target string) bool {
	for _, line := range lines {
		if line == target {
			return true
		}
	}
	return false
}

func containsSubstring(lines []string, target string) bool {
	for _, line := range lines {
		if strings.Contains(line, target) {
			return true
		}
	}
	return false
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
