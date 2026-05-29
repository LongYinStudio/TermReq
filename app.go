package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type App struct {
	stdin  *os.File
	stdout *os.File
}

func NewApp(stdin, stdout *os.File) *App {
	return &App{stdin: stdin, stdout: stdout}
}

func (a *App) Run() error {
	historyPath, historyPathErr := defaultHistoryPath()
	historyEntries, historyLoadErr := loadHistory(historyPath)

	model := newUIModelWithHistory(historyPath, historyEntries)
	switch {
	case historyPathErr != nil:
		model.message = fmt.Sprintf("History unavailable: %v", historyPathErr)
		model.messageLevel = "error"
	case historyLoadErr != nil:
		model.message = fmt.Sprintf("History unavailable: %v", historyLoadErr)
		model.messageLevel = "error"
	}

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(a.stdin),
		tea.WithOutput(a.stdout),
	)
	_, err := program.Run()
	return err
}

type focusTarget int

const (
	focusMethod focusTarget = iota
	focusURL
	focusTimeout
	focusHeaderKey
	focusHeaderValue
	focusBody
	focusHistory
	focusResponse
)

const (
	appChromeRows      = 4
	wideLayoutMinWidth = 118
	panelGap           = 1
	requestFixedRows   = 8
	responseFixedRows  = 2
	bodyMinRows        = 4
	bodyMaxRows        = 8
)

type requestDoneMsg struct {
	result  requestResult
	request RequestSpec
}

type uiKeyMap struct {
	NextFocus         key.Binding
	PrevFocus         key.Binding
	Send              key.Binding
	Quit              key.Binding
	MethodNav         key.Binding
	HeaderRowNav      key.Binding
	HeaderFieldSwitch key.Binding
	AddHeader         key.Binding
	DeleteHeader      key.Binding
	ResponseNav       key.Binding
	FormatJSON        key.Binding
	PasteCurl         key.Binding
	ExportCurl        key.Binding
	HistoryNav        key.Binding
	ApplyHistory      key.Binding
	DeleteHistory     key.Binding
}

func newUIKeyMap() uiKeyMap {
	return uiKeyMap{
		NextFocus:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
		PrevFocus:         key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
		Send:              key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "send")),
		Quit:              key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		MethodNav:         key.NewBinding(key.WithKeys("left", "right", "h", "l"), key.WithHelp("←/→", "method")),
		HeaderRowNav:      key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑/↓", "row")),
		HeaderFieldSwitch: key.NewBinding(key.WithKeys("left", "right", "h", "l"), key.WithHelp("←/→", "key/value")),
		AddHeader:         key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "add header")),
		DeleteHeader:      key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "drop header")),
		ResponseNav:       key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown", "home", "end"), key.WithHelp("pgup/dn", "scroll")),
		FormatJSON:        key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "format json")),
		PasteCurl:         key.NewBinding(key.WithKeys("paste cURL"), key.WithHelp("paste cURL", "import")),
		ExportCurl:        key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "export cURL")),
		HistoryNav:        key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑/↓", "history")),
		ApplyHistory:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "load history")),
		DeleteHistory:     key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "drop history")),
	}
}

type headerRow struct {
	keyInput   textinput.Model
	valueInput textinput.Model
}

type uiLayout struct {
	stacked                bool
	requestWidth           int
	requestHeight          int
	responseWidth          int
	responseHeight         int
	requestContentWidth    int
	requestContentHeight   int
	responseContentWidth   int
	responseContentHeight  int
	headerRowsVisible      int
	bodyHeight             int
	responseViewportHeight int
	headerKeyWidth         int
	headerValueWidth       int
}

type uiModel struct {
	width  int
	height int
	ready  bool

	focus       focusTarget
	methods     []string
	methodIndex int

	urlInput          textinput.Model
	timeoutInput      textinput.Model
	headerRows        []headerRow
	selectedHeaderRow int
	bodyInput         textarea.Model
	responseView      viewport.Model
	help              help.Model
	keys              uiKeyMap
	historyPath       string
	historyEntries    []HistoryEntry
	selectedHistory   int

	message      string
	messageLevel string
	sending      bool
	response     ResponseView

	layout uiLayout
}

func newUIModel() uiModel {
	return newUIModelWithHistory("", nil)
}

func newUIModelWithHistory(historyPath string, historyEntries []HistoryEntry) uiModel {
	urlInput := newTextInput("https://httpbin.org/get")
	timeoutInput := newTextInput("30s")
	bodyInput := newTextArea("")
	bodyInput.Placeholder = "{\n  \"hello\": \"world\"\n}"

	responseView := viewport.New(0, 0)
	responseView.MouseWheelEnabled = true

	model := uiModel{
		focus:             focusURL,
		methods:           []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		methodIndex:       0,
		urlInput:          urlInput,
		timeoutInput:      timeoutInput,
		headerRows:        []headerRow{newHeaderRow("Accept", "application/json")},
		selectedHeaderRow: 0,
		bodyInput:         bodyInput,
		responseView:      responseView,
		help:              newHelpModel(),
		keys:              newUIKeyMap(),
		historyPath:       historyPath,
		historyEntries:    append([]HistoryEntry(nil), historyEntries...),
		selectedHistory:   0,
		message:           "Ready. Paste a browser cURL anywhere to import, Ctrl+S sends.",
		messageLevel:      "info",
		response:          placeholderResponse(),
	}

	model.normalizeHistorySelection()
	model.syncFocus()
	return model
}

func newTextInput(value string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.SetValue(value)
	input.CursorEnd()
	input.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	input.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	return input
}

func newHeaderInput(value, placeholder string) textinput.Model {
	input := newTextInput(value)
	input.Placeholder = placeholder
	return input
}

func newHeaderRow(keyValue, valueValue string) headerRow {
	return headerRow{
		keyInput:   newHeaderInput(keyValue, "Header-Name"),
		valueInput: newHeaderInput(valueValue, "value"),
	}
}

func newTextArea(value string) textarea.Model {
	area := textarea.New()
	area.Prompt = ""
	area.ShowLineNumbers = false
	area.SetValue(value)
	area.FocusedStyle.Base = lipgloss.NewStyle()
	area.FocusedStyle.CursorLine = lipgloss.NewStyle()
	area.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	area.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	area.BlurredStyle.Base = lipgloss.NewStyle()
	area.BlurredStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	area.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	area.BlurredStyle.CursorLine = lipgloss.NewStyle()
	area.EndOfBufferCharacter = ' '
	area.SetCursor(0)
	return area
}

func (m uiModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.applyLayout()
		return m, nil

	case requestDoneMsg:
		m.sending = false
		m.response = msg.result.view
		m.message = msg.result.message
		m.messageLevel = msg.result.level
		if err := m.appendHistory(msg.request); err != nil {
			m.message = fmt.Sprintf("%s (history save failed: %v)", msg.result.message, err)
			m.messageLevel = "error"
		}
		m.rebuildResponseContent()
		return m, nil

	case tea.KeyMsg:
		if msg.Paste {
			if handled := m.handleCurlPaste(string(msg.Runes)); handled {
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextFocus):
			m.moveFocus(1)
			return m, nil
		case key.Matches(msg, m.keys.PrevFocus):
			m.moveFocus(-1)
			return m, nil
		case key.Matches(msg, m.keys.Send):
			return m.handleSend()
		case key.Matches(msg, m.keys.ExportCurl):
			return m.handleExportCurl()
		}

		if handled := m.handleFocusedKey(msg); handled {
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focus {
	case focusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case focusTimeout:
		m.timeoutInput, cmd = m.timeoutInput.Update(msg)
	case focusHeaderKey:
		row := m.currentHeaderRow()
		row.keyInput, cmd = row.keyInput.Update(msg)
		m.headerRows[m.selectedHeaderRow] = row
	case focusHeaderValue:
		row := m.currentHeaderRow()
		row.valueInput, cmd = row.valueInput.Update(msg)
		m.headerRows[m.selectedHeaderRow] = row
	case focusBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	case focusHistory:
	case focusResponse:
		m.responseView, cmd = m.responseView.Update(msg)
	}

	return m, cmd
}

func (m *uiModel) handleSend() (tea.Model, tea.Cmd) {
	if m.sending {
		m.message = "Request already in flight."
		m.messageLevel = "info"
		return m, nil
	}

	if _, err := parseTimeoutValue(m.timeoutInput.Value()); err != nil {
		m.response = requestErrorResult(err).view
		m.message = err.Error()
		m.messageLevel = "error"
		m.rebuildResponseContent()
		return m, nil
	}

	m.sending = true
	m.message = fmt.Sprintf("Sending %s %s", m.methods[m.methodIndex], strings.TrimSpace(m.urlInput.Value()))
	m.messageLevel = "info"

	request := m.currentRequestSpec()

	return m, requestCmd(
		request,
	)
}

func (m *uiModel) handleExportCurl() (tea.Model, tea.Cmd) {
	request := m.currentRequestSpec()
	curlCmd := exportCurlCommand(request)

	if err := copyToClipboard(curlCmd); err != nil {
		m.message = fmt.Sprintf("Export failed: %v", err)
		m.messageLevel = "error"
		return m, nil
	}

	m.response = ResponseView{
		Status:      "Exported",
		Proto:       "-",
		Duration:    "-",
		Size:        formatBytes(len(curlCmd)),
		HeaderLines: []string{},
		BodyLabel:   "cURL",
		BodyLines:   splitLines(curlCmd),
	}
	m.rebuildResponseContent()
	m.message = "Copied cURL command to clipboard."
	m.messageLevel = "success"
	return m, nil
}

func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func (m *uiModel) currentRequestSpec() RequestSpec {
	headers := make([]headerPair, 0, len(m.headerRows))
	for _, row := range m.headerRows {
		keyValue := strings.TrimSpace(row.keyInput.Value())
		valueValue := strings.TrimSpace(row.valueInput.Value())
		if keyValue == "" && valueValue == "" {
			continue
		}
		headers = append(headers, headerPair{Key: keyValue, Value: valueValue})
	}

	return normalizeRequestSpec(RequestSpec{
		Method:  m.methods[m.methodIndex],
		URL:     m.urlInput.Value(),
		Timeout: m.timeoutInput.Value(),
		Headers: headers,
		Body:    m.bodyInput.Value(),
	})
}

func (m *uiModel) applyRequestSpec(request RequestSpec) {
	request = normalizeRequestSpec(request)

	m.setMethod(request.Method)
	m.urlInput.SetValue(request.URL)
	m.urlInput.CursorEnd()
	timeoutValue := request.Timeout
	if timeoutValue == "" {
		timeoutValue = defaultTimeout.String()
	}
	m.timeoutInput.SetValue(timeoutValue)
	m.timeoutInput.CursorEnd()
	m.bodyInput.SetValue(request.Body)

	if len(request.Headers) == 0 {
		m.headerRows = []headerRow{newHeaderRow("", "")}
	} else {
		m.headerRows = make([]headerRow, 0, len(request.Headers))
		for _, header := range request.Headers {
			m.headerRows = append(m.headerRows, newHeaderRow(header.Key, header.Value))
		}
	}
	m.selectedHeaderRow = 0
	m.applyLayout()
	m.syncFocus()
}

func (m *uiModel) appendHistory(request RequestSpec) error {
	request = normalizeRequestSpec(request)
	if isEmptyRequestSpec(request) {
		return nil
	}

	entry := HistoryEntry{
		SavedAt: time.Now(),
		Request: request,
	}

	entries := make([]HistoryEntry, 0, len(m.historyEntries)+1)
	entries = append(entries, entry)
	for _, existing := range m.historyEntries {
		if requestSpecsEqual(existing.Request, request) {
			continue
		}
		entries = append(entries, existing)
		if len(entries) >= maxHistoryEntries {
			break
		}
	}

	m.historyEntries = entries
	m.selectedHistory = 0
	m.applyLayout()
	return saveHistory(m.historyPath, m.historyEntries)
}

func (m *uiModel) applySelectedHistory() bool {
	if len(m.historyEntries) == 0 {
		m.message = "History is empty."
		m.messageLevel = "info"
		return true
	}

	m.normalizeHistorySelection()
	entry := m.historyEntries[m.selectedHistory]
	m.applyRequestSpec(entry.Request)
	m.message = fmt.Sprintf(
		"Loaded history: %s %s",
		entry.Request.Method,
		truncateText(entry.Request.URL, 72),
	)
	m.messageLevel = "success"
	return true
}

func (m *uiModel) deleteSelectedHistory() bool {
	if len(m.historyEntries) == 0 {
		m.message = "History is already empty."
		m.messageLevel = "info"
		return true
	}

	m.normalizeHistorySelection()
	removed := m.historyEntries[m.selectedHistory]
	m.historyEntries = append(m.historyEntries[:m.selectedHistory], m.historyEntries[m.selectedHistory+1:]...)
	m.normalizeHistorySelection()
	m.applyLayout()

	if err := saveHistory(m.historyPath, m.historyEntries); err != nil {
		m.message = fmt.Sprintf("History delete failed: %v", err)
		m.messageLevel = "error"
		return true
	}

	m.message = fmt.Sprintf(
		"Deleted history: %s %s",
		removed.Request.Method,
		truncateText(removed.Request.URL, 72),
	)
	m.messageLevel = "success"
	return true
}

func (m *uiModel) normalizeHistorySelection() {
	if len(m.historyEntries) == 0 {
		m.selectedHistory = 0
		return
	}
	if m.selectedHistory < 0 {
		m.selectedHistory = 0
	}
	if m.selectedHistory >= len(m.historyEntries) {
		m.selectedHistory = len(m.historyEntries) - 1
	}
}

func (m *uiModel) moveHistorySelection(delta int) {
	if len(m.historyEntries) == 0 {
		return
	}
	m.selectedHistory += delta
	m.normalizeHistorySelection()
}

func requestCmd(request RequestSpec) tea.Cmd {
	headers := make([]string, 0, len(request.Headers))
	for _, header := range request.Headers {
		headers = append(headers, fmt.Sprintf("%s: %s", header.Key, header.Value))
	}

	timeout, _ := parseTimeoutValue(request.Timeout)
	return func() tea.Msg {
		return requestDoneMsg{
			result: performRequest(
				request.Method,
				request.URL,
				timeout,
				headers,
				request.Body,
			),
			request: request,
		}
	}
}

func (m *uiModel) handleCurlPaste(raw string) bool {
	if !looksLikeCurlPaste(raw) {
		return false
	}

	imported, err := parseCurlCommand(raw)
	if err != nil {
		m.message = err.Error()
		m.messageLevel = "error"
		return true
	}

	m.applyCurlImport(imported)
	return true
}

func (m *uiModel) applyCurlImport(imported curlImport) {
	method := strings.ToUpper(strings.TrimSpace(imported.Method))
	if method == "" {
		method = "GET"
	}
	m.setMethod(method)

	m.urlInput.SetValue(imported.URL)
	m.urlInput.CursorEnd()
	m.bodyInput.SetValue(imported.Body)

	if len(imported.Headers) == 0 {
		m.headerRows = []headerRow{newHeaderRow("", "")}
	} else {
		m.headerRows = make([]headerRow, 0, len(imported.Headers))
		for _, header := range imported.Headers {
			m.headerRows = append(m.headerRows, newHeaderRow(header.Key, header.Value))
		}
	}
	m.selectedHeaderRow = 0

	m.message = fmt.Sprintf(
		"Imported cURL: %s %s (%s, body %d bytes).",
		m.methods[m.methodIndex],
		truncateText(imported.URL, 72),
		pluralize(len(imported.Headers), "header", "headers"),
		len(imported.Body),
	)
	m.messageLevel = "success"

	m.applyLayout()
	m.syncFocus()
}

func (m *uiModel) setMethod(method string) {
	for index, candidate := range m.methods {
		if strings.EqualFold(candidate, method) {
			m.methodIndex = index
			return
		}
	}
	m.methods = append(m.methods, method)
	m.methodIndex = len(m.methods) - 1
}

func (m *uiModel) handleFocusedKey(msg tea.KeyMsg) bool {
	switch m.focus {
	case focusMethod:
		switch msg.String() {
		case "left", "h":
			m.methodIndex = (m.methodIndex - 1 + len(m.methods)) % len(m.methods)
			return true
		case "right", "l":
			m.methodIndex = (m.methodIndex + 1) % len(m.methods)
			return true
		default:
			typed := strings.ToUpper(msg.String())
			if len(typed) == 1 {
				for index, method := range m.methods {
					if strings.HasPrefix(method, typed) {
						m.methodIndex = index
						return true
					}
				}
			}
		}

	case focusHeaderKey, focusHeaderValue:
		switch {
		case key.Matches(msg, m.keys.AddHeader):
			m.addHeaderRow()
			return true
		case key.Matches(msg, m.keys.DeleteHeader):
			m.deleteHeaderRow()
			return true
		}

		switch msg.String() {
		case "up", "k":
			m.moveHeaderRow(-1)
			return true
		case "down", "j":
			m.moveHeaderRow(1)
			return true
		case "left", "h":
			if m.focus == focusHeaderValue {
				m.focus = focusHeaderKey
				m.syncFocus()
				return true
			}
		case "right", "l":
			if m.focus == focusHeaderKey {
				m.focus = focusHeaderValue
				m.syncFocus()
				return true
			}
		}

	case focusBody:
		if key.Matches(msg, m.keys.FormatJSON) {
			body := strings.TrimSpace(m.bodyInput.Value())
			if body == "" {
				m.message = "Body is empty."
				m.messageLevel = "info"
				return true
			}
			pretty, ok := prettyJSON([]byte(body))
			if !ok {
				m.message = "Body is not valid JSON."
				m.messageLevel = "error"
				return true
			}
			m.bodyInput.SetValue(pretty)
			m.message = "Body formatted as JSON."
			m.messageLevel = "success"
			return true
		}

	case focusHistory:
		switch {
		case key.Matches(msg, m.keys.DeleteHistory):
			return m.deleteSelectedHistory()
		case key.Matches(msg, m.keys.ApplyHistory):
			return m.applySelectedHistory()
		}

		switch msg.String() {
		case "up", "k":
			m.moveHistorySelection(-1)
			return true
		case "down", "j":
			m.moveHistorySelection(1)
			return true
		}
	}

	return false
}

func (m *uiModel) moveFocus(delta int) {
	target := (int(m.focus) + delta + int(focusResponse) + 1) % (int(focusResponse) + 1)
	m.focus = focusTarget(target)
	m.syncFocus()
}

func (m *uiModel) syncFocus() {
	m.urlInput.Blur()
	m.timeoutInput.Blur()
	m.bodyInput.Blur()
	for index := range m.headerRows {
		m.headerRows[index].keyInput.Blur()
		m.headerRows[index].valueInput.Blur()
	}

	switch m.focus {
	case focusURL:
		m.urlInput.Focus()
	case focusTimeout:
		m.timeoutInput.Focus()
	case focusHeaderKey:
		row := m.currentHeaderRow()
		row.keyInput.Focus()
		m.headerRows[m.selectedHeaderRow] = row
	case focusHeaderValue:
		row := m.currentHeaderRow()
		row.valueInput.Focus()
		m.headerRows[m.selectedHeaderRow] = row
	case focusBody:
		m.bodyInput.Focus()
	case focusHistory, focusResponse:
	}
}

func (m *uiModel) currentHeaderRow() headerRow {
	if len(m.headerRows) == 0 {
		m.headerRows = []headerRow{newHeaderRow("", "")}
		m.selectedHeaderRow = 0
	}
	if m.selectedHeaderRow < 0 {
		m.selectedHeaderRow = 0
	}
	if m.selectedHeaderRow >= len(m.headerRows) {
		m.selectedHeaderRow = len(m.headerRows) - 1
	}
	return m.headerRows[m.selectedHeaderRow]
}

func (m *uiModel) moveHeaderRow(delta int) {
	if len(m.headerRows) == 0 {
		return
	}
	next := m.selectedHeaderRow + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.headerRows) {
		next = len(m.headerRows) - 1
	}
	m.selectedHeaderRow = next
	m.syncFocus()
}

func (m *uiModel) addHeaderRow() {
	insertAt := m.selectedHeaderRow + 1
	if insertAt < 0 || insertAt > len(m.headerRows) {
		insertAt = len(m.headerRows)
	}

	m.headerRows = append(m.headerRows, headerRow{})
	copy(m.headerRows[insertAt+1:], m.headerRows[insertAt:])
	m.headerRows[insertAt] = newHeaderRow("", "")
	m.selectedHeaderRow = insertAt
	m.applyLayout()
	m.syncFocus()
}

func (m *uiModel) deleteHeaderRow() {
	if len(m.headerRows) == 0 {
		m.headerRows = []headerRow{newHeaderRow("", "")}
		m.selectedHeaderRow = 0
		m.syncFocus()
		return
	}

	if len(m.headerRows) == 1 {
		m.headerRows[0] = newHeaderRow("", "")
		m.selectedHeaderRow = 0
		m.syncFocus()
		return
	}

	m.headerRows = append(m.headerRows[:m.selectedHeaderRow], m.headerRows[m.selectedHeaderRow+1:]...)
	if m.selectedHeaderRow >= len(m.headerRows) {
		m.selectedHeaderRow = len(m.headerRows) - 1
	}
	m.applyLayout()
	m.syncFocus()
}

func (m *uiModel) headerLines() []string {
	lines := make([]string, 0, len(m.headerRows))
	for _, row := range m.headerRows {
		keyValue := strings.TrimSpace(row.keyInput.Value())
		valueValue := strings.TrimSpace(row.valueInput.Value())
		if keyValue == "" && valueValue == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", keyValue, valueValue))
	}
	return lines
}

func (m *uiModel) applyLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	m.layout = calculateLayout(m.width, m.height, len(m.headerRows))

	m.urlInput.Width = max(18, m.layout.requestContentWidth-12)
	m.timeoutInput.Width = max(10, m.layout.requestContentWidth-12)
	m.bodyInput.SetWidth(max(20, m.layout.requestContentWidth))
	m.bodyInput.SetHeight(m.layout.bodyHeight)

	for index := range m.headerRows {
		m.headerRows[index].keyInput.Width = m.layout.headerKeyWidth - 2
		m.headerRows[index].valueInput.Width = m.layout.headerValueWidth - 2
	}

	m.responseView.Width = m.layout.responseContentWidth
	m.responseView.Height = m.layout.responseViewportHeight
	m.help.Width = max(0, m.width-2)

	m.rebuildResponseContent()
}

func calculateLayout(width, height, headerCount int) uiLayout {
	panelFrameWidth := styles.panel.GetHorizontalFrameSize()
	panelFrameHeight := styles.panel.GetVerticalFrameSize()
	contentHeight := max(8, height-appChromeRows)

	layout := uiLayout{stacked: width < wideLayoutMinWidth}
	if layout.stacked {
		layout.requestWidth = width
		layout.responseWidth = width
		minRequestHeight := panelFrameHeight + requestFixedRows + 3
		minResponseHeight := panelFrameHeight + responseFixedRows + 4
		layout.requestHeight = contentHeight * 62 / 100
		if layout.requestHeight < minRequestHeight {
			layout.requestHeight = minRequestHeight
		}
		if contentHeight-layout.requestHeight < minResponseHeight {
			layout.requestHeight = max(1, contentHeight-minResponseHeight)
		}
		layout.responseHeight = max(1, contentHeight-layout.requestHeight)
	} else {
		layout.requestWidth = max(50, width*44/100)
		layout.responseWidth = max(50, width-layout.requestWidth-panelGap)
		layout.requestHeight = contentHeight
		layout.responseHeight = contentHeight
	}

	layout.requestContentWidth = max(24, layout.requestWidth-panelFrameWidth)
	layout.requestContentHeight = max(1, layout.requestHeight-panelFrameHeight)
	layout.responseContentWidth = max(24, layout.responseWidth-panelFrameWidth)
	layout.responseContentHeight = max(1, layout.responseHeight-panelFrameHeight)

	available := max(1, layout.requestContentHeight-requestFixedRows)
	bodyFloor := min(bodyMinRows, max(1, available-1))
	preferredHeaderRows := min(max(1, headerCount), 6)
	maxHeaderRows := max(1, available-bodyFloor)
	layout.headerRowsVisible = min(preferredHeaderRows, maxHeaderRows)
	layout.bodyHeight = min(max(1, available-layout.headerRowsVisible), bodyMaxRows)
	layout.responseViewportHeight = max(1, layout.responseContentHeight-responseFixedRows)

	rowIndexWidth := 3
	headerContentWidth := max(20, layout.requestContentWidth-rowIndexWidth-2)
	layout.headerKeyWidth = max(12, headerContentWidth*34/100)
	layout.headerValueWidth = max(12, headerContentWidth-layout.headerKeyWidth)

	return layout
}

func (m *uiModel) rebuildResponseContent() {
	if m.responseView.Width <= 0 {
		return
	}

	width := max(20, m.responseView.Width)
	headerMeta := pluralize(len(m.response.HeaderLines), "header", "headers")
	bodyMeta := pluralize(len(m.response.BodyLines), "line", "lines")
	lines := []string{
		sectionLineMeta("Headers", headerMeta, width),
	}

	if len(m.response.HeaderLines) == 0 {
		lines = append(lines, styles.muted.Render("  <empty>"))
	} else {
		for _, line := range m.response.HeaderLines {
			lines = append(lines, styles.text.Render("  "+line))
		}
	}

	lines = append(lines, "", sectionLineMeta(m.response.BodyLabel, bodyMeta, width))
	for _, line := range m.response.BodyLines {
		lines = append(lines, styles.text.Render("  "+line))
	}

	m.responseView.SetContent(strings.Join(lines, "\n"))
	m.responseView.GotoTop()
}

func (m uiModel) View() string {
	if !m.ready {
		return styles.app.Render("Loading...")
	}
	if m.width < 80 || m.height < 24 {
		return styles.app.Render("Terminal too small. Minimum size: 80x24.")
	}

	header := m.renderHeader()
	message := m.renderMessage()
	shortcuts := m.renderShortcutStrip()
	requestPane := m.renderRequestPane()
	responsePane := m.renderResponsePane()

	var body string
	if m.layout.stacked {
		body = lipgloss.JoinVertical(lipgloss.Left, requestPane, responsePane)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, requestPane, strings.Repeat(" ", panelGap), responsePane)
	}

	footer := styles.helpBar.Width(max(0, m.width-styles.helpBar.GetHorizontalFrameSize())).Render(m.help.View(m))
	return styles.app.Width(m.width).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, message, shortcuts, body, footer),
	)
}

func (m uiModel) renderHeader() string {
	state := "IDLE"
	stateView := styles.headerValue.Render(state)
	if m.sending {
		state = "SENDING"
		stateView = styles.messageInfo.Render(state)
	}
	parts := []string{
		styles.headerBadge.Render("TermReq"),
	}

	if m.width >= 96 {
		parts = append(parts, styles.headerSubtle.Render("terminal HTTP client"))
	}

	parts = append(parts,
		styles.headerLabel.Render("METHOD"),
		methodLipgloss(m.methods[m.methodIndex]).Render(m.methods[m.methodIndex]),
	)

	if m.width >= 110 {
		parts = append(parts,
			styles.headerLabel.Render("URL"),
			styles.headerValue.Render(truncateText(strings.TrimSpace(m.urlInput.Value()), max(16, m.width/4))),
		)
	}

	parts = append(parts,
		styles.headerLabel.Render("FOCUS"),
		styles.headerValue.Render(strings.ToUpper(m.focusLabel())),
		styles.headerLabel.Render("STATE"),
		stateView,
	)
	return styles.headerBar.Width(max(0, m.width-styles.headerBar.GetHorizontalFrameSize())).Render(strings.Join(parts, "  "))
}

func (m uiModel) renderMessage() string {
	indicator := messageLipgloss(m.messageLevel).Render("●")
	text := messageLipgloss(m.messageLevel).Render(m.message)
	return styles.messageBar.
		Width(max(0, m.width-styles.messageBar.GetHorizontalFrameSize())).
		Render(indicator + " " + text)
}

func (m uiModel) renderShortcutStrip() string {
	items := []string{
		styles.headerLabel.Render("KEYS"),
		renderShortcut("tab", "focus"),
		renderShortcut("shift+tab", "back"),
		renderShortcut("ctrl+s", "send"),
		renderShortcut("paste cURL", "import"),
		renderShortcut("ctrl+c", "quit"),
	}

	if m.width >= 100 {
		items = append(items, renderShortcut("ctrl+e", "export"))
	}

	if m.width >= 112 {
		items = append(items, m.renderFocusShortcutHints()...)
	}

	line := strings.Join(items, "  ")
	return styles.helpBar.
		Width(max(0, m.width-styles.helpBar.GetHorizontalFrameSize())).
		Render(line)
}

func (m uiModel) renderFocusShortcutHints() []string {
	switch m.focus {
	case focusMethod:
		return []string{renderShortcut("left/right", "method")}
	case focusHeaderKey, focusHeaderValue:
		return []string{
			renderShortcut("up/down", "row"),
			renderShortcut("left/right", "key/value"),
			renderShortcut("ctrl+n", "add"),
			renderShortcut("ctrl+d", "delete"),
		}
	case focusBody:
		return []string{renderShortcut("ctrl+p", "format JSON")}
	case focusHistory:
		return []string{
			renderShortcut("up/down", "history"),
			renderShortcut("enter", "load"),
			renderShortcut("ctrl+d", "delete"),
		}
	case focusResponse:
		return []string{renderShortcut("pgup/pgdn", "scroll")}
	default:
		return nil
	}
}

func renderShortcut(keyText, desc string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.headerValue.Render(keyText),
		" ",
		styles.hint.Render(desc),
	)
}

func (m uiModel) renderRequestPane() string {
	panelStyle := styles.panel
	active := m.focus != focusResponse && m.focus != focusHistory
	if active {
		panelStyle = styles.panelActive
	}
	detail := "Tab to edit request"
	if active {
		detail = "editing " + m.focusLabel()
	}

	rows := []string{
		m.renderPaneTitle("REQUEST", active, detail, m.layout.requestContentWidth),
		m.renderMethodRow(m.layout.requestContentWidth),
		m.renderInputRow("URL", m.urlInput.View(), m.focus == focusURL, m.layout.requestContentWidth),
		m.renderInputRow("TIMEOUT", m.timeoutInput.View(), m.focus == focusTimeout, m.layout.requestContentWidth),
		m.renderHeaderEditor(),
		sectionLineMeta("Body", bodySummary(m.bodyInput.Value()), m.layout.requestContentWidth),
		m.renderArea(m.bodyInput.View(), m.focus == focusBody, m.layout.requestContentWidth),
	}

	content := lipgloss.NewStyle().
		Width(m.layout.requestContentWidth).
		MaxWidth(m.layout.requestContentWidth).
		Height(m.layout.requestContentHeight).
		MaxHeight(m.layout.requestContentHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return panelStyle.Render(content)
}

func (m uiModel) renderResponsePane() string {
	panelStyle := styles.panel
	active := m.focus == focusResponse || m.focus == focusHistory
	if active {
		panelStyle = styles.panelActive
	}
	detail := "Tab to response"
	if m.focus == focusHistory {
		detail = "browse saved requests"
	} else if active {
		detail = "scroll response"
	}

	title := "RESPONSE"
	var body string
	if m.focus == focusHistory {
		title = "HISTORY"
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderPaneTitle(title, active, detail, m.layout.responseContentWidth),
			m.renderHistorySummary(m.layout.responseContentWidth),
			m.renderHistoryBrowser(),
		)
	} else {
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderPaneTitle(title, active, detail, m.layout.responseContentWidth),
			m.renderResponseSummary(m.layout.responseContentWidth),
			m.responseView.View(),
		)
	}

	content := lipgloss.NewStyle().
		Width(m.layout.responseContentWidth).
		MaxWidth(m.layout.responseContentWidth).
		Height(m.layout.responseContentHeight).
		MaxHeight(m.layout.responseContentHeight).
		Render(body)

	return panelStyle.Render(content)
}

func (m uiModel) renderResponseSummary(width int) string {
	status := truncateText(m.response.Status, max(8, width/3))
	metrics := []string{
		renderMetric("STATUS", status, statusLipgloss(m.response.Status)),
		renderMetric("TIME", m.response.Duration, styles.metricValue),
		renderMetric("SIZE", m.response.Size, styles.metricValue),
		renderMetric("PROTO", m.response.Proto, styles.metricValue),
	}
	if width < 56 {
		metrics = metrics[:3]
	}
	if width < 42 {
		metrics = metrics[:2]
	}
	if width < 30 {
		metrics = metrics[:1]
	}

	line := strings.Join(metrics, "  ")
	return styles.field.
		Width(max(10, width-styles.field.GetHorizontalFrameSize())).
		MaxWidth(max(10, width-styles.field.GetHorizontalFrameSize())).
		Render(line)
}

func renderMetric(label, value string, valueStyle lipgloss.Style) string {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.metricLabel.Render(label),
		" ",
		valueStyle.Render(value),
	)
}

func (m uiModel) renderPaneTitle(title string, active bool, detail string, width int) string {
	titleStyle := styles.panelTitle
	if active {
		titleStyle = styles.panelTitleActive
	}

	titleView := titleStyle.Render(title)
	detailWidth := max(0, width-lipgloss.Width(titleView)-1)
	detailView := styles.hint.Render(truncateText(detail, detailWidth))
	return lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, titleView, " ", detailView))
}

func (m uiModel) renderMethodRow(width int) string {
	active := m.focus == focusMethod
	rowStyle := styles.field
	if active {
		rowStyle = styles.fieldActive
	}

	label := styles.label.Render("METHOD")
	if active {
		label = styles.labelActive.Render("METHOD")
	}

	chips := make([]string, 0, len(m.methods))
	for index, method := range m.methods {
		chipStyle := styles.methodChip
		if index == m.methodIndex {
			chipStyle = methodLipgloss(method)
		}
		chips = append(chips, chipStyle.Render(method))
	}
	selector := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(chips, " "))
	contentWidth := max(10, width-rowStyle.GetHorizontalFrameSize())

	line := lipgloss.JoinHorizontal(lipgloss.Top, label, " ", selector)
	if lipgloss.Width(line) > contentWidth {
		value := methodLipgloss(m.methods[m.methodIndex]).Render(m.methods[m.methodIndex])
		hint := styles.hint.Render("  left/right cycle")
		line = lipgloss.JoinHorizontal(lipgloss.Top, label, " ", value, hint)
	}

	return rowStyle.
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(line)
}

func (m uiModel) renderInputRow(label, content string, active bool, width int) string {
	rowStyle := styles.field
	if active {
		rowStyle = styles.fieldActive
	}
	labelView := styles.label.Render(label)
	if active {
		labelView = styles.labelActive.Render(label)
	}
	contentWidth := max(10, width-rowStyle.GetHorizontalFrameSize())
	valueWidth := max(10, contentWidth-lipgloss.Width(labelView)-1)
	value := lipgloss.NewStyle().Width(valueWidth).Render(content)
	return rowStyle.
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, labelView, " ", value))
}

func (m uiModel) renderHeaderEditor() string {
	offset := headerViewportStart(m.selectedHeaderRow, m.layout.headerRowsVisible, len(m.headerRows))
	end := min(len(m.headerRows), offset+m.layout.headerRowsVisible)
	lines := []string{
		sectionLineMeta("Headers", headerEditorSummary(offset, end, len(m.headerRows)), m.layout.requestContentWidth),
		m.renderHeaderColumns(),
	}

	for i := 0; i < m.layout.headerRowsVisible; i++ {
		rowIndex := offset + i
		if rowIndex >= len(m.headerRows) {
			showHint := rowIndex == len(m.headerRows) && m.headerFocusActive()
			lines = append(lines, m.renderEmptyHeaderRow(showHint))
			continue
		}
		lines = append(lines, m.renderHeaderRow(rowIndex))
	}

	lines = append(lines, styles.hint.Render("Ctrl+N add row  Ctrl+D delete row"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m uiModel) renderHeaderColumns() string {
	index := styles.rowIndex.Render(" # ")
	keyLabel := styles.tableHeader.Width(m.layout.headerKeyWidth).Render("KEY")
	valueLabel := styles.tableHeader.Width(m.layout.headerValueWidth).Render("VALUE")
	return lipgloss.JoinHorizontal(lipgloss.Top, index, " ", keyLabel, " ", valueLabel)
}

func (m uiModel) renderHeaderRow(index int) string {
	row := m.headerRows[index]
	selected := index == m.selectedHeaderRow && m.headerFocusActive()
	keyActive := selected && m.focus == focusHeaderKey
	valueActive := selected && m.focus == focusHeaderValue

	indexStyle := styles.rowIndex
	if selected {
		indexStyle = styles.rowIndexActive
	}

	keyStyle := styles.field
	valueStyle := styles.field
	if selected {
		keyStyle = styles.fieldSelected
		valueStyle = styles.fieldSelected
	}
	if keyActive {
		keyStyle = styles.fieldActive
	}
	if valueActive {
		valueStyle = styles.fieldActive
	}

	keyCell := keyStyle.Width(m.layout.headerKeyWidth).MaxWidth(m.layout.headerKeyWidth).Render(row.keyInput.View())
	valueCell := valueStyle.Width(m.layout.headerValueWidth).MaxWidth(m.layout.headerValueWidth).Render(row.valueInput.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		indexStyle.Render(fmt.Sprintf("%2d ", index+1)),
		" ",
		keyCell,
		" ",
		valueCell,
	)
}

func (m uiModel) renderEmptyHeaderRow(showHint bool) string {
	index := styles.rowIndex.Render("   ")
	keyText := ""
	valueText := ""
	if showHint {
		keyText = styles.muted.Render("new header name")
		valueText = styles.muted.Render("value")
	}

	keyCell := styles.field.Width(m.layout.headerKeyWidth).MaxWidth(m.layout.headerKeyWidth).Render(keyText)
	valueCell := styles.field.Width(m.layout.headerValueWidth).MaxWidth(m.layout.headerValueWidth).Render(valueText)
	return lipgloss.JoinHorizontal(lipgloss.Top, index, " ", keyCell, " ", valueCell)
}

func (m uiModel) headerFocusActive() bool {
	return m.focus == focusHeaderKey || m.focus == focusHeaderValue
}

func (m uiModel) renderArea(content string, active bool, width int) string {
	wrap := styles.textareaWrap
	if active {
		wrap = styles.textareaActive
	}
	return wrap.Width(width).MaxWidth(width).Render(content)
}

func (m uiModel) renderHistorySummary(width int) string {
	summary := "No saved requests"
	if len(m.historyEntries) > 0 {
		selected := m.selectedHistory
		if selected < 0 {
			selected = 0
		}
		if selected >= len(m.historyEntries) {
			selected = len(m.historyEntries) - 1
		}
		entry := m.historyEntries[selected]
		summary = fmt.Sprintf(
			"%s  timeout %s  %s",
			entry.Request.Method,
			fallbackHistoryValue(entry.Request.Timeout, defaultTimeout.String()),
			entry.SavedAt.Local().Format("2006-01-02 15:04"),
		)
	}
	return styles.field.
		Width(max(10, width-styles.field.GetHorizontalFrameSize())).
		MaxWidth(max(10, width-styles.field.GetHorizontalFrameSize())).
		Render(truncateText(summary, max(10, width-styles.field.GetHorizontalFrameSize())))
}

func (m uiModel) renderHistoryBrowser() string {
	visibleRows := max(1, m.layout.responseContentHeight-3)
	offset := headerViewportStart(m.selectedHistory, visibleRows, len(m.historyEntries))
	end := min(len(m.historyEntries), offset+visibleRows)
	lines := []string{
		sectionLineMeta("Saved Requests", historySummary(offset, end, len(m.historyEntries)), m.layout.responseContentWidth),
	}

	if len(m.historyEntries) == 0 {
		lines = append(lines, styles.muted.Render("  No saved requests yet. Send a request to capture it here."))
	} else {
		for index := offset; index < end; index++ {
			lines = append(lines, m.renderHistoryRow(index, m.layout.responseContentWidth))
		}
	}

	lines = append(lines, styles.hint.Render("Enter load  Ctrl+D delete  Up/Down navigate"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m uiModel) renderHistoryRow(index, width int) string {
	entry := m.historyEntries[index]
	selected := index == m.selectedHistory
	active := selected && m.focus == focusHistory

	rowStyle := styles.field
	if selected {
		rowStyle = styles.fieldSelected
	}
	if active {
		rowStyle = styles.fieldActive
	}

	contentWidth := max(10, width-rowStyle.GetHorizontalFrameSize())
	marker := " "
	if selected {
		marker = ">"
	}
	markerPart := styles.headerValue.Render(marker)
	datePart := styles.hint.Render(entry.SavedAt.Local().Format("01-02 15:04"))
	methodPart := methodLipgloss(entry.Request.Method).Render(entry.Request.Method)
	urlWidth := max(6, contentWidth-lipgloss.Width(markerPart)-lipgloss.Width(datePart)-lipgloss.Width(methodPart)-3)
	urlPart := styles.text.Render(truncateText(entry.Request.URL, urlWidth))

	line := lipgloss.JoinHorizontal(lipgloss.Top, markerPart, " ", datePart, " ", methodPart, " ", urlPart)
	return rowStyle.Width(contentWidth).MaxWidth(contentWidth).Render(line)
}

func fallbackHistoryValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func headerEditorSummary(start, end, total int) string {
	if total == 0 {
		return "0 rows"
	}
	if start == 0 && end >= total {
		return pluralize(total, "row", "rows")
	}
	return fmt.Sprintf("%d-%d/%d rows", start+1, end, total)
}

func historySummary(start, end, total int) string {
	if total == 0 {
		return "empty"
	}
	if start == 0 && end >= total {
		return pluralize(total, "entry", "entries")
	}
	return fmt.Sprintf("%d-%d/%d entries", start+1, end, total)
}

func bodySummary(body string) string {
	if strings.TrimSpace(body) == "" {
		return "empty"
	}
	return fmt.Sprintf("%s, %d bytes", pluralize(lineCount(body), "line", "lines"), len(body))
}

func lineCount(value string) int {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func truncateText(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	suffix := "..."
	var builder strings.Builder
	for _, r := range value {
		next := builder.String() + string(r) + suffix
		if lipgloss.Width(next) > maxWidth {
			break
		}
		builder.WriteRune(r)
	}
	return builder.String() + suffix
}

func (m uiModel) focusLabel() string {
	switch m.focus {
	case focusMethod:
		return "method"
	case focusURL:
		return "url"
	case focusTimeout:
		return "timeout"
	case focusHeaderKey:
		return "header key"
	case focusHeaderValue:
		return "header value"
	case focusBody:
		return "body"
	case focusHistory:
		return "history"
	case focusResponse:
		return "response"
	default:
		return "unknown"
	}
}

func (m uiModel) ShortHelp() []key.Binding {
	bindings := []key.Binding{
		m.keys.NextFocus,
		m.keys.PrevFocus,
		m.keys.Send,
		m.keys.PasteCurl,
		m.keys.Quit,
	}

	switch m.focus {
	case focusMethod:
		bindings = append(bindings, m.keys.MethodNav)
	case focusHeaderKey, focusHeaderValue:
		bindings = append(bindings, m.keys.HeaderRowNav, m.keys.HeaderFieldSwitch, m.keys.AddHeader, m.keys.DeleteHeader)
	case focusBody:
		bindings = append(bindings, m.keys.FormatJSON)
	case focusHistory:
		bindings = append(bindings, m.keys.HistoryNav, m.keys.ApplyHistory, m.keys.DeleteHistory)
	case focusResponse:
		bindings = append(bindings, m.keys.ResponseNav)
	}

	return bindings
}

func (m uiModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func headerViewportStart(cursor, visible, total int) int {
	if total <= visible {
		return 0
	}
	start := cursor - visible/2
	if start < 0 {
		return 0
	}
	if start > total-visible {
		return total - visible
	}
	return start
}
