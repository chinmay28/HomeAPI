package display

import (
	"math"
	"strconv"
	"strings"
)

// NA is what a number slot shows when its symbol is missing from the quotes.
const NA = "N/A"

// cleanNumber strips the decoration a price string may carry (spaces,
// thousands commas, a leading currency sign) and returns the bare number
// text, or "" when nothing numeric is left.
func cleanNumber(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimPrefix(s, "$")
	return s
}

// sourceDecimals counts the digits after the decimal point in a bare number.
func sourceDecimals(s string) int {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return len(s) - i - 1
	}
	return 0
}

// parsePrice parses a quote's price string. It returns ok=false when the
// string is not a number at all.
func parsePrice(s string) (v float64, decimals int, ok bool) {
	c := cleanNumber(s)
	if c == "" {
		return 0, 0, false
	}
	v, err := strconv.ParseFloat(c, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, 0, false
	}
	return v, sourceDecimals(c), true
}

// FormatPrice renders a quote's price string for the panel.
//
// decimals caps how many decimals are shown; the source string is trimmed to
// it and never padded with precision the feed does not carry (nil keeps the
// source's own decimals). thousands inserts separators into the integer
// part. prefix goes immediately before the digits, after any minus sign. A
// string that is not a number is returned verbatim behind the prefix, so a
// feed that writes "halted" still shows something.
func FormatPrice(src string, decimals *int, prefix string, thousands bool) string {
	v, srcDec, ok := parsePrice(src)
	if !ok {
		return prefix + strings.TrimSpace(src)
	}
	d := srcDec
	if decimals != nil && *decimals < d {
		d = *decimals
	}
	return formatFloat(v, d, prefix, thousands)
}

// FormatComputed renders a number the server computed (a ratio): exactly
// decimals places, no separators.
func FormatComputed(v float64, decimals int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return NA
	}
	return formatFloat(v, decimals, "", false)
}

func formatFloat(v float64, decimals int, prefix string, thousands bool) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', decimals, 64)
	if thousands {
		s = groupThousands(s)
	}
	if v < 0 && strings.Trim(s, "0.") != "" {
		return "-" + prefix + s
	}
	return prefix + s
}

// groupThousands inserts commas into the integer part of a bare decimal.
func groupThousands(s string) string {
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	if len(intPart) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(intPart) % 3
	if lead > 0 {
		b.WriteString(intPart[:lead])
	}
	for i := lead; i < len(intPart); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(intPart[i : i+3])
	}
	return b.String() + frac
}

// firstLine returns the first line of s, trimmed, cut to at most max runes.
func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}
