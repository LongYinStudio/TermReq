package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHeaderLinesSkipsEmptyRows(t *testing.T) {
	model := newUIModel()
	model.headerRows = []headerRow{
		newHeaderRow("Accept", "application/json"),
		newHeaderRow("", ""),
		newHeaderRow("X-Trace", "abc"),
	}

	lines := model.headerLines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 header lines, got %d", len(lines))
	}
	if lines[0] != "Accept: application/json" {
		t.Fatalf("unexpected first header line: %q", lines[0])
	}
	if lines[1] != "X-Trace: abc" {
		t.Fatalf("unexpected second header line: %q", lines[1])
	}
}

func TestCalculateLayout(t *testing.T) {
	layout := calculateLayout(160, 40, 3)
	if layout.stacked {
		t.Fatalf("expected wide layout to be split")
	}
	if layout.requestWidth <= 0 || layout.responseWidth <= 0 {
		t.Fatalf("expected positive panel widths: %+v", layout)
	}
	if layout.bodyHeight <= 0 || layout.headerRowsVisible <= 0 {
		t.Fatalf("expected positive editor heights: %+v", layout)
	}

	stackedLayout := calculateLayout(100, 40, 3)
	if !stackedLayout.stacked {
		t.Fatalf("expected narrow layout to be stacked")
	}
	if stackedLayout.requestHeight <= 0 || stackedLayout.responseHeight <= 0 {
		t.Fatalf("expected positive stacked heights: %+v", stackedLayout)
	}
}

func TestCalculateLayoutContentFitsAllocatedHeight(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{width: 80, height: 24},
		{width: 100, height: 32},
		{width: 118, height: 24},
		{width: 160, height: 40},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			layout := calculateLayout(size.width, size.height, 8)
			if layout.stacked && layout.requestHeight+layout.responseHeight > size.height-appChromeRows {
				t.Fatalf("stacked panes exceed available height: %+v", layout)
			}

			requestRows := requestFixedRows + layout.headerRowsVisible + layout.bodyHeight
			if requestRows > layout.requestContentHeight {
				t.Fatalf("request content exceeds allocated height: rows=%d layout=%+v", requestRows, layout)
			}
			if layout.bodyHeight > bodyMaxRows {
				t.Fatalf("body editor exceeds visual cap: %+v", layout)
			}

			responseRows := responseFixedRows + layout.responseViewportHeight
			if responseRows > layout.responseContentHeight {
				t.Fatalf("response content exceeds allocated height: rows=%d layout=%+v", responseRows, layout)
			}
		})
	}
}

func TestHistorySelectionLoadsRequestIntoEditor(t *testing.T) {
	model := newUIModelWithHistory("", []HistoryEntry{
		{
			Request: RequestSpec{
				Method:  "POST",
				URL:     "https://api.example.test/widgets",
				Timeout: "5s",
				Headers: []headerPair{{Key: "Authorization", Value: "Bearer token"}},
				Body:    `{"name":"demo"}`,
			},
		},
	})
	model.focus = focusHistory

	if handled := model.applySelectedHistory(); !handled {
		t.Fatalf("expected history load to be handled")
	}
	if got := model.methods[model.methodIndex]; got != "POST" {
		t.Fatalf("expected method POST, got %q", got)
	}
	if got := model.urlInput.Value(); got != "https://api.example.test/widgets" {
		t.Fatalf("unexpected URL: %q", got)
	}
	if got := model.timeoutInput.Value(); got != "5s" {
		t.Fatalf("unexpected timeout: %q", got)
	}
	if got := model.bodyInput.Value(); got != `{"name":"demo"}` {
		t.Fatalf("unexpected body: %q", got)
	}
	if len(model.headerRows) != 1 || model.headerRows[0].keyInput.Value() != "Authorization" {
		t.Fatalf("expected imported header row, got %+v", model.headerRows)
	}
}

func TestHistoryDeleteRemovesEntry(t *testing.T) {
	model := newUIModelWithHistory("", []HistoryEntry{
		{Request: RequestSpec{Method: "GET", URL: "https://example.com/one"}},
		{Request: RequestSpec{Method: "GET", URL: "https://example.com/two"}},
	})
	model.selectedHistory = 1

	if handled := model.deleteSelectedHistory(); !handled {
		t.Fatalf("expected history delete to be handled")
	}
	if len(model.historyEntries) != 1 {
		t.Fatalf("expected one history entry, got %d", len(model.historyEntries))
	}
	if got := model.historyEntries[0].Request.URL; got != "https://example.com/one" {
		t.Fatalf("unexpected remaining history URL: %q", got)
	}
}

func TestViewShowsShortcutStrip(t *testing.T) {
	model := newUIModel()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 140, Height: 32})
	view := updated.(uiModel).View()

	for _, want := range []string{"KEYS", "ctrl+s", "paste cURL", "ctrl+c"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to include shortcut %q", want)
		}
	}
}

func TestViewFitsTerminalBounds(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{width: 80, height: 24},
		{width: 100, height: 32},
		{width: 118, height: 24},
		{width: 160, height: 40},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			model := newUIModel()
			updated, _ := model.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			view := updated.(uiModel).View()

			if height := lipgloss.Height(view); height > size.height {
				t.Fatalf("view exceeds terminal height: got %d want <= %d", height, size.height)
			}
			for lineNumber, line := range strings.Split(view, "\n") {
				if lineWidth := lipgloss.Width(line); lineWidth > size.width {
					t.Fatalf("line %d exceeds terminal width: got %d want <= %d", lineNumber+1, lineWidth, size.width)
				}
			}
		})
	}
}
