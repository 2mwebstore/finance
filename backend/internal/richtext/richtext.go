// Package richtext keeps descriptions to a small, safe subset of HTML.
//
// The output is rebuilt from parsed tokens (never copied through), so only the
// allowed tags, plus http/https/mailto links, can survive. Everything else is
// dropped while its text is kept; script/style content is removed entirely.
package richtext

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// MaxHTMLLength caps a sanitized description.
const MaxHTMLLength = 8000

var allowedTags = map[atom.Atom]bool{
	atom.B: true, atom.Strong: true, atom.I: true, atom.Em: true, atom.U: true, atom.S: true,
	atom.P: true, atom.Div: true, atom.Br: true, atom.Ul: true, atom.Ol: true, atom.Li: true, atom.A: true,
}

// Content of these elements is never shown.
var droppedContent = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Template: true, atom.Noscript: true,
	atom.Iframe: true, atom.Object: true, atom.Textarea: true, atom.Title: true,
}

var blockTags = map[atom.Atom]bool{atom.P: true, atom.Div: true, atom.Li: true, atom.Br: true, atom.Ul: true, atom.Ol: true}

// Sanitize returns the allowed subset of in. Plain text (no tags) is returned escaped.
func Sanitize(in string) string {
	z := xhtml.NewTokenizer(strings.NewReader(in))
	var b strings.Builder
	var open []atom.Atom
	skip := 0
	for {
		tt := z.Next()
		switch tt {
		case xhtml.ErrorToken:
			for i := len(open) - 1; i >= 0; i-- {
				b.WriteString("</" + open[i].String() + ">")
			}
			return strings.TrimSpace(b.String())
		case xhtml.TextToken:
			if skip == 0 {
				b.WriteString(html.EscapeString(string(z.Text())))
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			tok := z.Token()
			if droppedContent[tok.DataAtom] {
				if tt == xhtml.StartTagToken {
					skip++
				}
				continue
			}
			if skip > 0 || !allowedTags[tok.DataAtom] {
				continue
			}
			name := tok.DataAtom.String()
			if tok.DataAtom == atom.A {
				href := safeHref(tok.Attr)
				if href == "" {
					continue // keep the link text, drop the link
				}
				b.WriteString(`<a href="` + html.EscapeString(href) + `" target="_blank" rel="noopener noreferrer nofollow">`)
			} else {
				b.WriteString("<" + name + ">")
			}
			if tok.DataAtom == atom.Br {
				continue
			}
			if tt == xhtml.SelfClosingTagToken {
				b.WriteString("</" + name + ">")
				continue
			}
			open = append(open, tok.DataAtom)
		case xhtml.EndTagToken:
			tok := z.Token()
			if droppedContent[tok.DataAtom] {
				if skip > 0 {
					skip--
				}
				continue
			}
			if skip > 0 {
				continue
			}
			for i := len(open) - 1; i >= 0; i-- {
				if open[i] == tok.DataAtom {
					for j := len(open) - 1; j >= i; j-- {
						b.WriteString("</" + open[j].String() + ">")
					}
					open = open[:i]
					break
				}
			}
		}
	}
}

func safeHref(attrs []xhtml.Attribute) string {
	for _, a := range attrs {
		if strings.ToLower(a.Key) != "href" {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(a.Val))
		if err != nil {
			return ""
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https", "mailto":
			return u.String()
		}
		return ""
	}
	return ""
}

var spaces = regexp.MustCompile(`[ \t\r\f\v]+`)
var blankLines = regexp.MustCompile(`\n\s*\n+`)

// ToText converts rich text to plain text (for CSV exports and short labels).
func ToText(in string) string {
	z := xhtml.NewTokenizer(strings.NewReader(in))
	var b strings.Builder
	skip := 0
	for {
		tt := z.Next()
		switch tt {
		case xhtml.ErrorToken:
			out := blankLines.ReplaceAllString(spaces.ReplaceAllString(b.String(), " "), "\n")
			lines := strings.Split(out, "\n")
			for i := range lines {
				lines[i] = strings.TrimSpace(lines[i])
			}
			return strings.TrimSpace(strings.Join(lines, "\n"))
		case xhtml.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken, xhtml.EndTagToken:
			name, _ := z.TagName()
			a := atom.Lookup(name)
			if droppedContent[a] {
				if tt == xhtml.StartTagToken {
					skip++
				} else if tt == xhtml.EndTagToken && skip > 0 {
					skip--
				}
				continue
			}
			if blockTags[a] {
				b.WriteString("\n")
			}
		}
	}
}
