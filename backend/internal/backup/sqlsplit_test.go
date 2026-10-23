package backup

import "testing"

func TestSplitStatementsRespectsQuotesAndComments(t *testing.T) {
	script := "-- header\n/*!40101 SET NAMES utf8mb4 */;\n" +
		"/* plain comment; ignored */\n" +
		"INSERT INTO `t` VALUES ('a;b','it\\'s','x''y',\"q;\");\n" +
		"# hash comment ;\n" +
		"CREATE TABLE `t` (`v` text);"
	got := splitStatements(script)
	want := []string{
		"/*!40101 SET NAMES utf8mb4 */",
		"INSERT INTO `t` VALUES ('a;b','it\\'s','x''y',\"q;\")",
		"CREATE TABLE `t` (`v` text)",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d statements %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("statement %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCheckStatements(t *testing.T) {
	ok := splitStatements("SET NAMES utf8mb4; DROP TABLE IF EXISTS `t`; CREATE TABLE `t` (id int); INSERT INTO `t` VALUES (1); /*!40000 ALTER TABLE `t` ENABLE KEYS */;")
	if err := checkStatements(ok); err != nil {
		t.Fatalf("dump refused: %v", err)
	}
	for _, bad := range []string{
		"CREATE TABLE `t` (id int); DROP DATABASE prod;",
		"CREATE TABLE `t` (id int); GRANT ALL ON *.* TO 'x';",
		"CREATE TABLE `t` (id int); INSERT INTO `t` SELECT 1 INTO OUTFILE '/tmp/x';",
		"CREATE TABLE `t` (id int); CREATE USER 'x';",
		"INSERT INTO `t` VALUES (1);",
	} {
		if err := checkStatements(splitStatements(bad)); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestSQLLiteral(t *testing.T) {
	cases := []struct {
		v    any
		typ  string
		want string
	}{
		{nil, "VARCHAR", "NULL"},
		{int64(42), "INT", "42"},
		{[]byte("it's a\nline\\"), "TEXT", `'it\'s a\nline\\'`},
		{[]byte("12.5000"), "DECIMAL", "'12.5000'"},
		{[]byte{0x00, 0xff}, "BLOB", "0x00ff"},
	}
	for _, c := range cases {
		if got := sqlLiteral(c.v, c.typ); got != c.want {
			t.Errorf("sqlLiteral(%v, %s) = %s, want %s", c.v, c.typ, got, c.want)
		}
	}
	// What Dump writes splits back into the same statements.
	got := splitStatements("INSERT INTO `t` VALUES (" + sqlLiteral([]byte("a;b\n'c'"), "TEXT") + ");")
	if len(got) != 1 {
		t.Fatalf("escaped value split into %d statements: %q", len(got), got)
	}
}
