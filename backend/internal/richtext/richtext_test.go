package richtext

import "testing"

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"plain & simple":                                   "plain &amp; simple",
		"<b>Bold</b> and <i>it</i>":                         "<b>Bold</b> and <i>it</i>",
		`<p onclick="x()">Hi<script>alert(1)</script></p>`: "<p>Hi</p>",
		`<img src=x onerror=alert(1)>text`:                  "text",
		`<a href="javascript:alert(1)">bad</a>`:             "bad",
		`<a href="https://example.com" onclick="x">ok</a>`:  `<a href="https://example.com" target="_blank" rel="noopener noreferrer nofollow">ok</a>`,
		"<ul><li>one<li>two</ul>":                           "<ul><li>one<li>two</li></li></ul>",
		"<b>unclosed":                                       "<b>unclosed</b>",
		"line<br>break":                                     "line<br>break",
		`<style>p{}</style><u>u</u>`:                        "<u>u</u>",
		`"><svg/onload=alert(1)>`:                           "&#34;&gt;",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToText(t *testing.T) {
	in := "<p>Office <b>supplies</b></p><ul><li>Paper</li><li>Ink &amp; toner</li></ul><script>x</script>"
	want := "Office supplies\nPaper\nInk & toner"
	if got := ToText(in); got != want {
		t.Errorf("ToText = %q, want %q", got, want)
	}
}
