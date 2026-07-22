package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func colorizeJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s
	}
	if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(s), "", "  "); err == nil {
			s = pretty.String()
		}
	}

	var result strings.Builder
	afterColon := false
	var stack []byte

	inArray := func() bool {
		return len(stack) > 0 && stack[len(stack)-1] == '['
	}

	i := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case '"':
			end := findStringEnd(s, i+1)
			if end > i {
				str := s[i : end+1]
				if !afterColon && !inArray() {
					result.WriteString(jsonKeyStyle.Render(str))
				} else {
					result.WriteString(jsonStringStyle.Render(str))
				}
				i = end + 1
				afterColon = false
				continue
			}
		case ':':
			afterColon = true
			result.WriteByte(c)
			i++
			continue
		case ',':
			afterColon = false
			result.WriteByte(c)
			i++
			continue
		case '{', '[':
			stack = append(stack, c)
			afterColon = false
			result.WriteString(jsonBracketStyle.Render(string(c)))
			i++
			continue
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			afterColon = false
			result.WriteString(jsonBracketStyle.Render(string(c)))
			i++
			continue
		case 't', 'f', 'n':
			if token, ok := matchLiteral(s, i); ok {
				if token == "null" {
					result.WriteString(jsonNullStyle.Render(token))
				} else {
					result.WriteString(jsonBoolStyle.Render(token))
				}
				i += len(token)
				afterColon = false
				continue
			}
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if afterColon || inArray() {
				end := i + 1
				for end < len(s) && (isDigit(s[end]) || s[end] == '.' || s[end] == 'e' || s[end] == 'E' || s[end] == '+' || s[end] == '-') {
					end++
				}
				result.WriteString(jsonNumberStyle.Render(s[i:end]))
				i = end
				afterColon = false
				continue
			}
		}
		result.WriteByte(c)
		i++
	}
	return result.String()
}

func findStringEnd(s string, start int) int {
	escaped := false
	for i := start; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == '"' {
			return i
		}
	}
	return -1
}

func matchLiteral(s string, i int) (string, bool) {
	literals := []string{"true", "false", "null"}
	for _, lit := range literals {
		if strings.HasPrefix(s[i:], lit) {
			end := i + len(lit)
			if end == len(s) || !isIdentChar(s[end]) {
				return lit, true
			}
		}
	}
	return "", false
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isIdentChar(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}
