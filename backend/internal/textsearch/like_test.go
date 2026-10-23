package textsearch

import "testing"

func TestContainsEscapesWildcards(t *testing.T) {
	cases := map[string]string{
		"office":    "%office%",
		"  office ": "%office%",
		"50%":       `%50\%%`,
		"a_b":       `%a\_b%`,
		`c:\tmp`:    `%c:\\tmp%`,
	}
	for in, want := range cases {
		if got := Contains(in); got != want {
			t.Errorf("Contains(%q) = %q, want %q", in, got, want)
		}
	}
}
