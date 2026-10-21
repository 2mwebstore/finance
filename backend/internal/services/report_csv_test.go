package services

import "testing"

func TestCSVTextDefusesFormulas(t *testing.T) {
	cases := map[string]string{
		"Office supplies":   "Office supplies",
		"=HYPERLINK(\"x\")": "'=HYPERLINK(\"x\")",
		"+1":                "'+1",
		"-5":                "'-5",
		"@SUM(A1)":          "'@SUM(A1)",
		"":                  "",
	}
	for in, want := range cases {
		if got := csvText(in); got != want {
			t.Errorf("csvText(%q) = %q, want %q", in, got, want)
		}
	}
}
