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

// listWindow returns [start, end) for a list viewport.
// When the selection leaves the bottom of the visible page, the window jumps
// so the selection sits at the top of the new page (not sticky mid/bottom).
func listWindow(selected, total, maxVisible int) (start, end int) {
	if total <= 0 {
		return 0, 0
	}
	if maxVisible < 1 {
		maxVisible = 1
	}
	selected = clamp(selected, 0, total-1)
	if total <= maxVisible {
		return 0, total
	}
	// Page-based: selection index maps to a page whose first row is selection's page start.
	page := selected / maxVisible
	start = page * maxVisible
	end = min(start+maxVisible, total)
	// Last page may be short — still show trailing rows without shifting selection off-page.
	return start, end
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

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return string(r[0])
	}
	return string(r[:n-1]) + "…"
}

// wrapPlainLines hard-wraps plain text lines to width (rune-aware).
func wrapPlainLines(lines []string, width int) []string {
	if width < 8 {
		width = 8
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > width {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, string(runes))
	}
	return out
}

// detailContentWidth is usable text width inside the value box (excludes padding).
func detailContentWidth(boxWidth int) int {
	w := boxWidth - 4 // Padding(1, 2)
	if w < 20 {
		return 20
	}
	return w
}

const detailChromeLines = 16

// detailMaxVisible returns how many content lines fit in the value box.
func detailMaxVisible(height int) int {
	maxVisible := height - detailChromeLines
	if maxVisible < 5 {
		return 5
	}
	return maxVisible
}

// scrollValueLines windows lines into maxVisible slots, reserving space for scroll hints.
func scrollValueLines(valueLines []string, scroll, maxVisible int) (visible []string, topHint, bottomHint string, clampedScroll int) {
	if maxVisible < 1 {
		maxVisible = 1
	}
	clampedScroll = max(scroll, 0)
	total := len(valueLines)
	if total <= maxVisible {
		return valueLines, "", "", 0
	}

	contentRows := func(scrolled bool) int {
		n := maxVisible - 1
		if scrolled {
			n--
		}
		return max(n, 1)
	}

	avail := contentRows(clampedScroll > 0)
	maxScroll := total - avail
	clampedScroll = min(clampedScroll, maxScroll)
	avail = contentRows(clampedScroll > 0)
	end := min(clampedScroll+avail, total)

	if end >= total {
		rows := maxVisible
		if clampedScroll > 0 {
			rows--
		}
		end = min(clampedScroll+max(rows, 1), total)
	}

	visible = valueLines[clampedScroll:end]
	if clampedScroll > 0 {
		topHint = metaDimStyle.Render(fmt.Sprintf("↑ %d more lines above", clampedScroll))
	}
	if end < total {
		bottomHint = metaDimStyle.Render(fmt.Sprintf("↓ %d more lines below", total-end))
	}
	return visible, topHint, bottomHint, clampedScroll
}

// ensureDetailCursorVisible keeps DetailScroll so DetailCursor stays in the viewport.
func ensureDetailCursorVisible(cursor, scroll, total, maxVisible int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	window := maxVisible - 2
	if window < 1 {
		window = 1
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+window {
		scroll = cursor - window + 1
	}
	if scroll < 0 {
		scroll = 0
	}
	return cursor, scroll
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

func prettyJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] != '{' && s[0] != '[' {
		return s
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(s), "", "  "); err != nil {
		return s
	}
	return pretty.String()
}

func colorizeJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s
	}
	s = prettyJSON(s)
	return colorizeJSONFragment(s)
}

// colorizeJSONFragment colors JSON text without re-indenting (safe for single lines).
func colorizeJSONFragment(s string) string {
	if s == "" {
		return s
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
				// On a pretty-printed line, a key is usually `"key":` (after leading spaces).
				// Treat as key when not afterColon and not clearly a value in array.
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
			// Numbers after colon, in arrays, or standalone on a pretty line after ": ".
			if afterColon || inArray() || looksLikeJSONNumber(s, i) {
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

func looksLikeJSONNumber(s string, i int) bool {
	// True when the token is a number ending at comma/brace/newline (pretty line values).
	if i > 0 {
		prev := s[i-1]
		if prev != ' ' && prev != ':' && prev != '[' && prev != ',' {
			return false
		}
	}
	j := i
	if s[j] == '-' {
		j++
	}
	if j >= len(s) || !isDigit(s[j]) {
		return false
	}
	return true
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
