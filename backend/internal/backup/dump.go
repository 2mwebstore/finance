package backup

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// rowsPerInsert keeps each INSERT statement well under MySQL's default packet size.
const rowsPerInsert = 200

// Dump writes every table of the connected database to w as a SQL script (DROP, CREATE, INSERT),
// in pure Go so no mysqldump binary is needed on the server.
func Dump(ctx context.Context, db *sql.DB, w io.Writer) error {
	out := bufio.NewWriter(w)
	fmt.Fprintf(out, "-- Income & Expense database backup\n-- Created %s\n\n", time.Now().Format(time.RFC3339))
	out.WriteString("SET NAMES utf8mb4;\nSET FOREIGN_KEY_CHECKS=0;\nSET UNIQUE_CHECKS=0;\nSET SQL_MODE='NO_AUTO_VALUE_ON_ZERO';\n\n")

	tables, err := baseTables(ctx, db)
	if err != nil {
		return err
	}
	for _, table := range tables {
		if err := dumpTable(ctx, db, out, table); err != nil {
			return fmt.Errorf("table %s: %w", table, err)
		}
	}
	out.WriteString("SET FOREIGN_KEY_CHECKS=1;\nSET UNIQUE_CHECKS=1;\n")
	return out.Flush()
}

func baseTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW FULL TABLES WHERE Table_type = 'BASE TABLE'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func dumpTable(ctx context.Context, db *sql.DB, out *bufio.Writer, table string) error {
	var name, create string
	if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE "+quoteIdent(table)).Scan(&name, &create); err != nil {
		return err
	}
	fmt.Fprintf(out, "DROP TABLE IF EXISTS %s;\n%s;\n", quoteIdent(table), create)

	rows, err := db.QueryContext(ctx, "SELECT * FROM "+quoteIdent(table))
	if err != nil {
		return err
	}
	defer rows.Close()
	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	values := make([]any, len(types))
	ptrs := make([]any, len(types))
	for i := range values {
		ptrs[i] = &values[i]
	}
	count := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if count%rowsPerInsert == 0 {
			if count > 0 {
				out.WriteString(";\n")
			}
			fmt.Fprintf(out, "INSERT INTO %s VALUES ", quoteIdent(table))
		} else {
			out.WriteString(",")
		}
		out.WriteString("(")
		for i, v := range values {
			if i > 0 {
				out.WriteString(",")
			}
			out.WriteString(sqlLiteral(v, types[i].DatabaseTypeName()))
		}
		out.WriteString(")")
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		out.WriteString(";\n")
	}
	out.WriteString("\n")
	return nil
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

var literalEscaper = strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\x00", `\0`, "\n", `\n`, "\r", `\r`, "\x1a", `\Z`)

// sqlLiteral renders one value as MySQL accepts it back. Text is escaped (newlines too, so a
// statement never spans a raw newline inside a string); binary data becomes a hex literal.
func sqlLiteral(v any, dbType string) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case bool:
		if t {
			return "1"
		}
		return "0"
	case time.Time:
		if t.IsZero() {
			return "'0000-00-00 00:00:00'"
		}
		if dbType == "DATE" {
			return "'" + t.Format("2006-01-02") + "'"
		}
		return "'" + t.Format("2006-01-02 15:04:05.999999") + "'"
	case []byte:
		switch dbType {
		case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY", "BIT", "GEOMETRY":
			if len(t) == 0 {
				return "''"
			}
			return "0x" + hex.EncodeToString(t)
		}
		return "'" + literalEscaper.Replace(string(t)) + "'"
	case string:
		return "'" + literalEscaper.Replace(t) + "'"
	default:
		return "'" + literalEscaper.Replace(fmt.Sprint(t)) + "'"
	}
}

// Restore runs a backup script against the database on one connection (so the session settings
// in the script apply to every statement). Every statement is checked first: a file with anything
// a backup wouldn't contain is refused before the database is touched.
func Restore(ctx context.Context, db *sql.DB, script string) (int, error) {
	stmts := splitStatements(script)
	if err := checkStatements(stmts); err != nil {
		return 0, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return 0, err
	}
	for i, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return i, fmt.Errorf("statement %d of %d failed (%s): %w", i+1, len(stmts), preview(stmt), err)
		}
	}
	_, _ = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1")
	return len(stmts), nil
}

// Validate checks a script would be accepted by Restore, without running it.
func Validate(script string) error {
	return checkStatements(splitStatements(script))
}
