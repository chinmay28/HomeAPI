package display

import "testing"

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		decimals  *int
		prefix    string
		thousands bool
		want      string
	}{
		{"hero thousands, zero decimals", "109432.12", intp(0), "$", true, "$109,432"},
		{"hero without prefix", "109432", intp(0), "", true, "109,432"},
		{"tile two decimals as-is", "301.20", intp(2), "$", false, "$301.20"},
		{"trim to two decimals", "58.4123", intp(2), "$", false, "$58.41"},
		{"never pad to decimals", "61.3", intp(2), "$", false, "$61.3"},
		{"integer stays integer", "190", intp(2), "$", false, "$190"},
		{"nil decimals keeps source", "1.23456", nil, "", false, "1.23456"},
		{"source commas are tolerated", "1,234.5", intp(1), "$", true, "$1,234.5"},
		{"source currency sign is not doubled", "$42.00", intp(2), "$", false, "$42.00"},
		{"whitespace trimmed", "  7.5 ", intp(1), "", false, "7.5"},
		{"negative keeps sign before prefix", "-5.25", intp(2), "$", false, "-$5.25"},
		{"not a number passes through", "halted", intp(2), "$", false, "$halted"},
		{"empty", "", intp(2), "$", false, "$"},
		{"large with separators and decimals", "1234567.891", intp(2), "", true, "1,234,567.89"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPrice(tt.src, tt.decimals, tt.prefix, tt.thousands); got != tt.want {
				t.Errorf("FormatPrice(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestFormatComputed(t *testing.T) {
	tests := []struct {
		v        float64
		decimals int
		want     string
	}{
		{301.20 / 214.05, 2, "1.41"},
		{58.41 / 301.20, 2, "0.19"},
		{2, 2, "2.00"},
		{0.5, 0, "0"},
	}
	for _, tt := range tests {
		if got := FormatComputed(tt.v, tt.decimals); got != tt.want {
			t.Errorf("FormatComputed(%v, %d) = %q, want %q", tt.v, tt.decimals, got, tt.want)
		}
	}
}

func TestGroupThousands(t *testing.T) {
	tests := map[string]string{
		"0":           "0",
		"999":         "999",
		"1000":        "1,000",
		"109432":      "109,432",
		"1234567.89":  "1,234,567.89",
		"100.5":       "100.5",
		"12345678901": "12,345,678,901",
	}
	for in, want := range tests {
		if got := groupThousands(in); got != want {
			t.Errorf("groupThousands(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"42", "42"},
		{"  42  \n", "42"},
		{"hello\nworld", "hello"},
		{"hello\r\nworld", "hello"},
		{"12345678901234", "1234567890"},
		{"", ""},
		{"\n\nsecond", "second"}, // leading blank lines are not the first line
	}
	for _, tt := range tests {
		if got := firstLine(tt.in, sumMaxRunes); got != tt.want {
			t.Errorf("firstLine(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
