package backup

import (
	"fmt"
	"regexp"
	"strings"
)

// splitStatements cuts a SQL script into statements at semicolons that are outside quotes and
// comments. Line (--, #) and block (/* */) comments are dropped; MySQL's executable comments
// (/*!40101 ... */) are kept, since they carry the SET statements mysqldump writes.
func splitStatements(script string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	n := len(script)
	for i := 0; i < n; i++ {
		c := script[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			// Copy the quoted text as is; backslash escapes and doubled quotes stay inside it.
			cur.WriteByte(c)
			for i++; i < n; i++ {
				cur.WriteByte(script[i])
				if script[i] == '\\' && c != '`' && i+1 < n {
					i++
					cur.WriteByte(script[i])
					continue
				}
				if script[i] == c {
					if i+1 < n && script[i+1] == c {
						i++
						cur.WriteByte(script[i])
						continue
					}
					break
				}
			}
		case c == '-' && i+2 < n && script[i+1] == '-' && (script[i+2] == ' ' || script[i+2] == '\t' || script[i+2] == '\n' || script[i+2] == '\r'),
			c == '#':
			for i < n && script[i] != '\n' {
				i++
			}
			cur.WriteByte('\n')
		case c == '/' && i+1 < n && script[i+1] == '*':
			end := strings.Index(script[i+2:], "*/")
			if end < 0 {
				end = n - i - 2
			}
			body := script[i : i+2+end]
			if strings.HasPrefix(body, "/*!") {
				cur.WriteString(body)
				cur.WriteString("*/")
			}
			i += 2 + end + 1
		case c == ';':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// Only the statements a database dump needs may run during a restore. Anything else (another
// database, users, files on the server, stored programs) is refused before anything is changed.
var allowedStatement = regexp.MustCompile(`(?is)^(?:/\*!\d*\s*)?(?:` +
	`set\s|` +
	`drop\s+table\s|` +
	`create\s+table\s|` +
	`insert\s+(?:ignore\s+)?into\s|` +
	`replace\s+into\s|` +
	`lock\s+tables\s|` +
	`unlock\s+tables|` +
	`alter\s+table\s+\S+\s+(?:disable|enable)\s+keys|` +
	`start\s+transaction|begin|commit)`)

var (
	forbiddenInside = regexp.MustCompile(`(?is)\binto\s+(?:out|dump)file\b|\bload_file\s*\(`)
	createTable     = regexp.MustCompile(`(?is)^create\s+table\s`)
)

// checkStatements refuses a script with any statement a dump wouldn't contain.
func checkStatements(stmts []string) error {
	if len(stmts) == 0 {
		return fmt.Errorf("the file has no SQL statements")
	}
	creates := 0
	for i, s := range stmts {
		if !allowedStatement.MatchString(s) || forbiddenInside.MatchString(s) {
			return fmt.Errorf("statement %d is not allowed in a backup: %s", i+1, preview(s))
		}
		if createTable.MatchString(s) {
			creates++
		}
	}
	if creates == 0 {
		return fmt.Errorf("the file has no CREATE TABLE statements, so it isn't a database backup")
	}
	return nil
}

func preview(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return s
}
