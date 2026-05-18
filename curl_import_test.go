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
