package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseBrowserCurlCommand(t *testing.T) {
	raw := `curl 'https://example.com/api/items' \
  -H 'accept: application/json' \
  -H 'content-type: application/json' \
  --data-raw '{"hello":"world"}' \
  --compressed`

	imported, err := parseCurlCommand(raw)
	if err != nil {
		t.Fatalf("parse curl: %v", err)
	}
	if imported.Method != "POST" {
		t.Fatalf("expected POST from data option, got %q", imported.Method)
	}
	if imported.URL != "https://example.com/api/items" {
		t.Fatalf("unexpected URL: %q", imported.URL)
	}
	if imported.Body != `{"hello":"world"}` {
		t.Fatalf("unexpected body: %q", imported.Body)
	}
	if got := findImportedHeader(imported.Headers, "accept"); got != "application/json" {
		t.Fatalf("unexpected accept header: %q", got)
	}
	if got := findImportedHeader(imported.Headers, "content-type"); got != "application/json" {
		t.Fatalf("unexpected content-type header: %q", got)
	}
}

func TestParseCurlCommandWithRequestAndConvenienceFlags(t *testing.T) {
	raw := `curl "https://example.com/profile" -XPUT -H "X-Trace: abc" -b "sid=1" -A "Mozilla/5.0"`

	imported, err := parseCurlCommand(raw)
	if err != nil {
		t.Fatalf("parse curl: %v", err)
	}
	if imported.Method != "PUT" {
		t.Fatalf("expected PUT, got %q", imported.Method)
	}
	if got := findImportedHeader(imported.Headers, "x-trace"); got != "abc" {
		t.Fatalf("unexpected x-trace header: %q", got)
	}
	if got := findImportedHeader(imported.Headers, "cookie"); got != "sid=1" {
		t.Fatalf("unexpected cookie header: %q", got)
	}
	if got := findImportedHeader(imported.Headers, "user-agent"); got != "Mozilla/5.0" {
		t.Fatalf("unexpected user-agent header: %q", got)
	}
}

func TestParseWindowsBrowserCurlCommand(t *testing.T) {
	raw := "curl \"https://example.com/search\" ^\r\n  -H \"Accept: application/json\" ^\r\n  --data-raw \"q=term\""

	imported, err := parseCurlCommand(raw)
	if err != nil {
		t.Fatalf("parse curl: %v", err)
	}
	if imported.Method != "POST" {
		t.Fatalf("expected POST, got %q", imported.Method)
	}
	if imported.URL != "https://example.com/search" {
		t.Fatalf("unexpected URL: %q", imported.URL)
	}
	if imported.Body != "q=term" {
		t.Fatalf("unexpected body: %q", imported.Body)
	}
	if got := findImportedHeader(imported.Headers, "accept"); got != "application/json" {
		t.Fatalf("unexpected accept header: %q", got)
	}
}

func TestCurlPasteAppliesToUIModel(t *testing.T) {
	raw := `curl 'https://api.example.test/widgets' \
  -H 'Authorization: Bearer token' \
  -H 'Content-Type: application/json' \
  --data-raw '{"name":"demo"}'`

	model := newUIModel()
	updated, _ := model.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(raw),
		Paste: true,
	})
	got := updated.(uiModel)

	if got.methods[got.methodIndex] != "POST" {
		t.Fatalf("expected imported method POST, got %q", got.methods[got.methodIndex])
	}
	if got.urlInput.Value() != "https://api.example.test/widgets" {
		t.Fatalf("unexpected imported URL: %q", got.urlInput.Value())
	}
	if got.bodyInput.Value() != `{"name":"demo"}` {
		t.Fatalf("unexpected imported body: %q", got.bodyInput.Value())
	}
	if len(got.headerRows) != 2 {
		t.Fatalf("expected 2 imported headers, got %d", len(got.headerRows))
	}
	if got.headerRows[0].keyInput.Value() != "Authorization" || got.headerRows[0].valueInput.Value() != "Bearer token" {
		t.Fatalf("unexpected first imported header: %q=%q", got.headerRows[0].keyInput.Value(), got.headerRows[0].valueInput.Value())
	}
	if got.messageLevel != "success" {
		t.Fatalf("expected success import message, got %q", got.messageLevel)
	}
}

func TestNonCurlPasteStillUpdatesFocusedInput(t *testing.T) {
	model := newUIModel()
	model.urlInput.SetValue("")

	updated, _ := model.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("https://plain.example.test"),
		Paste: true,
	})
	got := updated.(uiModel)

	if got.urlInput.Value() != "https://plain.example.test" {
		t.Fatalf("expected non-curl paste to reach URL input, got %q", got.urlInput.Value())
	}
}

func findImportedHeader(headers []headerPair, key string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Key, key) {
			return header.Value
		}
	}
	return ""
}

func TestExportCurlCommandSimpleGet(t *testing.T) {
	spec := RequestSpec{
		Method:  "GET",
		URL:     "https://example.com/api/items",
		Timeout: "30s",
	}

	curlCmd := exportCurlCommand(spec)
	expected := "curl --max-time 30 https://example.com/api/items"
	if curlCmd != expected {
		t.Fatalf("expected %q, got %q", expected, curlCmd)
	}
}

func TestExportCurlCommandPostWithHeadersAndBody(t *testing.T) {
	spec := RequestSpec{
		Method:  "POST",
		URL:     "https://api.example.com/data",
		Timeout: "10s",
		Headers: []headerPair{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Authorization", Value: "Bearer token123"},
		},
		Body: `{"key":"value"}`,
	}

	curlCmd := exportCurlCommand(spec)
	t.Logf("Generated cURL: %s", curlCmd)

	if !strings.Contains(curlCmd, "curl -X POST") {
		t.Fatal("expected -X POST")
	}
	if !strings.Contains(curlCmd, "--max-time 10") {
		t.Fatal("expected --max-time 10")
	}
	if !strings.Contains(curlCmd, "-H 'Content-Type: application/json'") {
		t.Fatal("expected Content-Type header")
	}
	if !strings.Contains(curlCmd, "-H 'Authorization: Bearer token123'") {
		t.Fatal("expected Authorization header")
	}
	if !strings.Contains(curlCmd, "-d '{\"key\":\"value\"}'") {
		t.Fatal("expected -d with body")
	}
	if !strings.Contains(curlCmd, "https://api.example.com/data") {
		t.Fatal("expected URL")
	}
}

func TestExportCurlCommandGetWithoutTimeout(t *testing.T) {
	spec := RequestSpec{
		Method: "GET",
		URL:    "https://example.com",
	}

	curlCmd := exportCurlCommand(spec)
	if strings.Contains(curlCmd, "--max-time") {
		t.Fatal("expected no --max-time when timeout is empty")
	}
}

func TestExportCurlCommandURLWithSpecialChars(t *testing.T) {
	spec := RequestSpec{
		Method: "GET",
		URL:    "https://example.com/path?a=1&b=2",
	}

	curlCmd := exportCurlCommand(spec)
	if !strings.Contains(curlCmd, "https://example.com/path?a=1&b=2") {
		t.Fatal("expected URL with query params")
	}
}

func TestExportCurlCommandURLWithSpaces(t *testing.T) {
	spec := RequestSpec{
		Method: "GET",
		URL:    "https://example.com/path with spaces",
	}

	curlCmd := exportCurlCommand(spec)
	if !strings.Contains(curlCmd, "'https://example.com/path with spaces'") {
		t.Fatal("expected URL to be quoted")
	}
}

func TestExportCurlCommandRoundTrip(t *testing.T) {
	raw := `curl -X POST 'https://api.example.com/data' -H 'Content-Type: application/json' -d '{"hello":"world"}'`

	imported, err := parseCurlCommand(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	spec := RequestSpec{
		Method:  imported.Method,
		URL:     imported.URL,
		Timeout: "30s",
		Headers: imported.Headers,
		Body:    imported.Body,
	}

	exported := exportCurlCommand(spec)

	if !strings.Contains(exported, "-X POST") {
		t.Fatal("round trip lost method")
	}
	if !strings.Contains(exported, "https://api.example.com/data") {
		t.Fatal("round trip lost URL")
	}
	if !strings.Contains(exported, "-H 'Content-Type: application/json'") {
		t.Fatal("round trip lost header")
	}
	if !strings.Contains(exported, "-d") {
		t.Fatal("round trip lost body")
	}
}
