package main

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
)

type curlImport struct {
	Method  string
	URL     string
	Headers []headerPair
	Body    string
}

type headerPair struct {
	Key   string
	Value string
}

func looksLikeCurlPaste(raw string) bool {
	tokens, err := shellWords(raw)
	if err != nil {
		return startsWithCurlCommand(raw)
	}
	return findCurlToken(tokens) >= 0
}

func startsWithCurlCommand(raw string) bool {
	cleaned := strings.TrimSpace(strings.ToLower(raw))
	return cleaned == "curl" ||
		strings.HasPrefix(cleaned, "curl ") ||
		strings.HasPrefix(cleaned, "curl\t") ||
		strings.HasPrefix(cleaned, "curl.exe ") ||
		strings.HasPrefix(cleaned, "$ curl ")
}

func parseCurlCommand(raw string) (curlImport, error) {
	tokens, err := shellWords(raw)
	if err != nil {
		return curlImport{}, err
	}

	start := findCurlToken(tokens)
	if start < 0 {
		return curlImport{}, errors.New("pasted text is not a curl command")
	}

	tokens = tokens[start+1:]
	if len(tokens) == 0 {
		return curlImport{}, errors.New("curl command is missing a URL")
	}

	var result curlImport
	var dataParts []string
	explicitMethod := false
	useGet := false

	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if token == "" {
			continue
		}

		if token == "--" {
			if index+1 < len(tokens) && result.URL == "" {
				result.URL = tokens[index+1]
			}
			break
		}

		if !strings.HasPrefix(token, "-") || token == "-" {
			if result.URL == "" {
				result.URL = token
			}
			continue
		}

		switch {
		case token == "-X" || token == "--request":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a method", token)
			}
			result.Method = strings.ToUpper(strings.TrimSpace(value))
			explicitMethod = result.Method != ""

		case strings.HasPrefix(token, "-X") && token != "-X":
			result.Method = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(token, "-X")))
			explicitMethod = result.Method != ""

		case strings.HasPrefix(token, "--request="):
			result.Method = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(token, "--request=")))
			explicitMethod = result.Method != ""

		case token == "-H" || token == "--header":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a header", token)
			}
			header, err := parseCurlHeader(value)
			if err != nil {
				return curlImport{}, err
			}
			result.Headers = append(result.Headers, header)

		case strings.HasPrefix(token, "-H") && token != "-H":
			header, err := parseCurlHeader(strings.TrimPrefix(token, "-H"))
			if err != nil {
				return curlImport{}, err
			}
			result.Headers = append(result.Headers, header)

		case strings.HasPrefix(token, "--header="):
			header, err := parseCurlHeader(strings.TrimPrefix(token, "--header="))
			if err != nil {
				return curlImport{}, err
			}
			result.Headers = append(result.Headers, header)

		case token == "--url":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a URL", token)
			}
			result.URL = value

		case strings.HasPrefix(token, "--url="):
			result.URL = strings.TrimPrefix(token, "--url=")

		case isCurlDataOption(token):
			value, ok := optionValue(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires data", token)
			}
			dataParts = append(dataParts, value)

		case token == "-A" || token == "--user-agent":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a value", token)
			}
			result.Headers = upsertHeader(result.Headers, "User-Agent", value)

		case strings.HasPrefix(token, "-A") && token != "-A":
			result.Headers = upsertHeader(result.Headers, "User-Agent", strings.TrimPrefix(token, "-A"))

		case strings.HasPrefix(token, "--user-agent="):
			result.Headers = upsertHeader(result.Headers, "User-Agent", strings.TrimPrefix(token, "--user-agent="))

		case token == "-e" || token == "--referer":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a value", token)
			}
			result.Headers = upsertHeader(result.Headers, "Referer", value)

		case strings.HasPrefix(token, "--referer="):
			result.Headers = upsertHeader(result.Headers, "Referer", strings.TrimPrefix(token, "--referer="))

		case token == "-b" || token == "--cookie":
			value, ok := nextToken(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a cookie value", token)
			}
			result.Headers = upsertHeader(result.Headers, "Cookie", value)

		case strings.HasPrefix(token, "-b") && token != "-b":
			result.Headers = upsertHeader(result.Headers, "Cookie", strings.TrimPrefix(token, "-b"))

		case strings.HasPrefix(token, "--cookie="):
			result.Headers = upsertHeader(result.Headers, "Cookie", strings.TrimPrefix(token, "--cookie="))

		case token == "-I" || token == "--head":
			if !explicitMethod {
				result.Method = "HEAD"
			}

		case token == "-G" || token == "--get":
			useGet = true
			if !explicitMethod {
				result.Method = "GET"
			}

		case optionConsumesValue(token):
			_, ok := optionValue(tokens, &index, token)
			if !ok {
				return curlImport{}, fmt.Errorf("%s requires a value", token)
			}

		default:
			// Browser exports contain flags such as --compressed, --location,
			// --insecure, and --globoff. They do not change the editable request.
		}
	}

	result.URL = strings.TrimSpace(result.URL)
	if result.URL == "" {
		return curlImport{}, errors.New("curl command is missing a URL")
	}

	if len(dataParts) > 0 {
		body := strings.Join(dataParts, "&")
		if useGet {
			withQuery, err := appendQueryData(result.URL, body)
			if err != nil {
				return curlImport{}, err
			}
			result.URL = withQuery
		} else {
			result.Body = body
			if !explicitMethod && result.Method == "" {
				result.Method = "POST"
			}
		}
	}

	if result.Method == "" {
		result.Method = "GET"
	}

	return result, nil
}

func shellWords(raw string) ([]string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "^\n", " ")

	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	inANSIC := false
	tokenStarted := false

	flush := func() {
		if tokenStarted {
			tokens = append(tokens, current.String())
			current.Reset()
			tokenStarted = false
		}
	}

	for index := 0; index < len(raw); index++ {
		ch := raw[index]

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
				continue
			}
			current.WriteByte(ch)

		case inANSIC:
			if ch == '\'' {
				inANSIC = false
				continue
			}
			if ch == '\\' && index+1 < len(raw) {
				index++
				current.WriteString(decodeANSICEscape(raw[index]))
				continue
			}
			current.WriteByte(ch)

		case inDouble:
			if ch == '"' {
				inDouble = false
				continue
			}
			if ch == '\\' && index+1 < len(raw) {
				next := raw[index+1]
				index++
				if next == '\n' {
					continue
				}
				current.WriteByte(next)
				tokenStarted = true
				continue
			}
			current.WriteByte(ch)

		default:
			if unicode.IsSpace(rune(ch)) {
				flush()
				continue
			}
			tokenStarted = true
			switch ch {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '$':
				if index+1 < len(raw) && raw[index+1] == '\'' {
					inANSIC = true
					index++
				} else {
					current.WriteByte(ch)
				}
			case '\\':
				if index+1 < len(raw) {
					index++
					if raw[index] == '\n' {
						continue
					}
					current.WriteByte(raw[index])
				} else {
					current.WriteByte(ch)
				}
			default:
				current.WriteByte(ch)
			}
		}
	}

	if inSingle || inDouble || inANSIC {
		return nil, errors.New("curl command has an unterminated quote")
	}
	flush()
	return tokens, nil
}

func decodeANSICEscape(ch byte) string {
	switch ch {
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 't':
		return "\t"
	case 'b':
		return "\b"
	case 'f':
		return "\f"
	case '\\', '\'', '"':
		return string(ch)
	default:
		return string(ch)
	}
}

func findCurlToken(tokens []string) int {
	for index, token := range tokens {
		name := strings.ToLower(filepath.Base(token))
		if name == "curl" || name == "curl.exe" {
			return index
		}
	}
	return -1
}

func parseCurlHeader(raw string) (headerPair, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return headerPair{}, fmt.Errorf("invalid curl header %q, expected Key: Value", raw)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return headerPair{}, errors.New("curl header name cannot be empty")
	}
	return headerPair{Key: key, Value: strings.TrimSpace(parts[1])}, nil
}

func nextToken(tokens []string, index *int, flag string) (string, bool) {
	if *index+1 >= len(tokens) {
		return "", false
	}
	*index++
	return tokens[*index], true
}

func optionValue(tokens []string, index *int, token string) (string, bool) {
	if eq := strings.Index(token, "="); strings.HasPrefix(token, "--") && eq >= 0 {
		return token[eq+1:], true
	}
	for _, prefix := range []string{"-d", "--data-raw", "--data-binary", "--data-ascii", "--data-urlencode", "--data"} {
		if strings.HasPrefix(token, prefix) && token != prefix {
			if strings.HasPrefix(prefix, "--") {
				continue
			}
			return strings.TrimPrefix(token, prefix), true
		}
	}
	return nextToken(tokens, index, token)
}

func isCurlDataOption(token string) bool {
	switch token {
	case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii", "--data-urlencode":
		return true
	}
	for _, prefix := range []string{"-d", "--data=", "--data-raw=", "--data-binary=", "--data-ascii=", "--data-urlencode="} {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}
	return false
}

func optionConsumesValue(token string) bool {
	switch token {
	case "-o", "--output", "-u", "--user", "--connect-timeout", "--max-time", "--proxy", "--resolve":
		return true
	}
	return false
}

func upsertHeader(headers []headerPair, key, value string) []headerPair {
	for index := range headers {
		if strings.EqualFold(headers[index].Key, key) {
			headers[index].Value = value
			return headers
		}
	}
	return append(headers, headerPair{Key: key, Value: value})
}

func appendQueryData(rawURL, data string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid curl url: %w", err)
	}
	if parsed.RawQuery == "" {
		parsed.RawQuery = data
	} else {
		parsed.RawQuery += "&" + data
	}
	return parsed.String(), nil
}
