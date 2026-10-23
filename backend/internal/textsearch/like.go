// Package textsearch builds LIKE patterns for the list pages' search boxes.
package textsearch

import "strings"

var escaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Contains returns a LIKE pattern matching text that contains q. % and _ in q are matched as
// themselves, not as wildcards. Callers skip the filter when q is empty.
func Contains(q string) string {
	return "%" + escaper.Replace(strings.TrimSpace(q)) + "%"
}
