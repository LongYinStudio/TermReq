package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultTimeout  = 30 * time.Second
	maxResponseBody = 2 << 20
)

type ResponseView struct {
	Status      string
	Proto       string
	Duration    string
	Size        string
	HeaderLines []string
	BodyLines   []string
	BodyLabel   string
}

type requestResult struct {
	view    ResponseView
	message string
	level   string
}

func placeholderResponse() ResponseView {
	return ResponseView{
		Status:      "No request sent",
		Proto:       "-",
		Duration:    "-",
		Size:        "-",
		HeaderLines: []string{},
		BodyLabel:   "Preview",
		BodyLines: []string{
			"No response yet.",
			"",
			"Press Ctrl+S to send the current request.",
			"Move focus to Response to scroll large output.",
		},
	}
}

func performRequest(method, rawURL string, timeout time.Duration, headerLines []string, body string) requestResult {
	client := &http.Client{Timeout: timeout}
	return performRequestWithClient(client, method, rawURL, headerLines, body)
}

func performRequestWithClient(client *http.Client, method, rawURL string, headerLines []string, body string) requestResult {
	normalizedURL, err := normalizeURL(rawURL)
	if err != nil {
		return requestErrorResult(err)
	}

	headers, err := parseHeaderLines(headerLines)
	if err != nil {
		return requestErrorResult(err)
	}

	request, err := http.NewRequest(method, normalizedURL, strings.NewReader(body))
	if err != nil {
		return requestErrorResult(fmt.Errorf("build request: %w", err))
	}

	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	if request.Header.Get("User-Agent") == "" {
		request.Header.Set("User-Agent", "TermReq/1.0")
	}

	startedAt := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return requestErrorResult(fmt.Errorf("request failed: %w", err))
	}
	defer response.Body.Close()

	bodyBytes, truncated, err := readResponseBody(response.Body)
	if err != nil {
		return requestErrorResult(fmt.Errorf("read response: %w", err))
	}

	view := buildResponseView(response, bodyBytes, time.Since(startedAt), truncated)
	return requestResult{
		view:    view,
		message: fmt.Sprintf("%s %s -> %s", method, normalizedURL, response.Status),
		level:   "success",
	}
}

func requestErrorResult(err error) requestResult {
	return requestResult{
		view: ResponseView{
			Status:      "Request failed",
			Proto:       "-",
			Duration:    "-",
			Size:        "-",
			HeaderLines: []string{},
			BodyLabel:   "Error",
			BodyLines:   splitLines(err.Error()),
		},
		message: err.Error(),
		level:   "error",
	}
}

func normalizeURL(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "", errors.New("url is required")
	}
	if !strings.Contains(cleaned, "://") {
		cleaned = "https://" + cleaned
	}
	parsed, err := url.Parse(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("url must include a valid host")
	}
	return parsed.String(), nil
}

func parseTimeoutValue(raw string) (time.Duration, error) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return defaultTimeout, nil
	}
	if looksNumeric(cleaned) {
		cleaned += "s"
	}
	duration, err := time.ParseDuration(cleaned)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q", raw)
	}
	if duration <= 0 {
		return 0, errors.New("timeout must be greater than zero")
	}
	return duration, nil
}

func looksNumeric(value string) bool {
	dotCount := 0
	for _, r := range value {
		if r == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}

func parseHeaderLines(lines []string) (http.Header, error) {
	headers := make(http.Header)
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q, expected Key: Value", rawLine)
		}
		key := textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, errors.New("header name cannot be empty")
		}
		headers.Add(key, value)
	}
	return headers, nil
}

func readResponseBody(body io.Reader) ([]byte, bool, error) {
	limited := io.LimitReader(body, maxResponseBody+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	truncated := len(data) > maxResponseBody
	if truncated {
		data = data[:maxResponseBody]
	}
	return data, truncated, nil
}

func buildResponseView(response *http.Response, body []byte, duration time.Duration, truncated bool) ResponseView {
	headerLines := make([]string, 0, len(response.Header))
	keys := make([]string, 0, len(response.Header))
	for key := range response.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		headerLines = append(headerLines, fmt.Sprintf("%s: %s", key, strings.Join(response.Header.Values(key), ", ")))
	}

	bodyLines := formatResponseBody(body, response.Header.Get("Content-Type"))
	if truncated {
		bodyLines = append([]string{
			fmt.Sprintf("[response truncated at %s]", formatBytes(maxResponseBody)),
			"",
		}, bodyLines...)
	}

	return ResponseView{
		Status:      response.Status,
		Proto:       response.Proto,
		Duration:    humanDuration(duration),
		Size:        formatBytes(len(body)),
		HeaderLines: headerLines,
		BodyLines:   bodyLines,
		BodyLabel:   "Body",
	}
}

func formatResponseBody(body []byte, contentType string) []string {
	if len(body) == 0 {
		return []string{"<empty>"}
	}

	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if normalizedType == "" {
		normalizedType = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(body), ";")[0]))
	}

	if strings.Contains(normalizedType, "json") {
		if pretty, ok := prettyJSON(body); ok {
			return splitLines(pretty)
		}
	}

	if isTextContent(normalizedType, body) {
		return splitLines(string(body))
	}

	dump := strings.TrimRight(hex.Dump(body), "\n")
	lines := []string{
		fmt.Sprintf("[binary payload, %s]", formatBytes(len(body))),
		"",
	}
	lines = append(lines, splitLines(dump)...)
	return lines
}

func prettyJSON(body []byte) (string, bool) {
	var out bytes.Buffer
	if err := json.Indent(&out, body, "", "  "); err != nil {
		return "", false
	}
	return out.String(), true
}

func isTextContent(contentType string, body []byte) bool {
	if strings.HasPrefix(contentType, "text/") {
		return true
	}
	textTypes := []string{
		"application/xml",
		"application/xhtml+xml",
		"application/javascript",
		"application/x-www-form-urlencoded",
		"application/sql",
		"application/graphql-response+json",
	}
	for _, candidate := range textTypes {
		if strings.Contains(contentType, candidate) {
			return true
		}
	}
	if !utf8.Valid(body) {
		return false
	}
	for _, r := range string(body) {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

func splitLines(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return []string{"<empty>"}
	}
	return strings.Split(normalized, "\n")
}

func humanDuration(duration time.Duration) string {
	switch {
	case duration < time.Millisecond:
		return duration.Round(time.Microsecond).String()
	case duration < time.Second:
		return duration.Round(time.Millisecond).String()
	default:
		return duration.Round(10 * time.Millisecond).String()
	}
}

func formatBytes(size int) string {
	units := []string{"B", "KiB", "MiB", "GiB"}
	value := float64(size)
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}
	if unitIndex == 0 {
		return fmt.Sprintf("%d %s", size, units[unitIndex])
	}
	return fmt.Sprintf("%.1f %s", value, units[unitIndex])
}
