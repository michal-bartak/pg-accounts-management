package pg

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/michalbartak/dbaccounts/internal/calltemplate"
	"github.com/michalbartak/dbaccounts/internal/model"
)

var gucNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// Querier is the subset of pgx used to run an operation. Both *pgx.Conn and pgx.Tx satisfy it,
// so an operation can run either standalone (autocommit) or inside a per-cluster transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// quoteConfigValue renders a role-GUC value as a SQL string literal. A value with a
// backslash is emitted as an E'…' escape string (both backslashes and single quotes
// doubled) so the literal is safe even when the server has standard_conforming_strings off;
// backslash-free values keep the plain '…' form.
func quoteConfigValue(v string) string {
	if strings.Contains(v, `\`) {
		return `E'` + strings.NewReplacer(`\`, `\\`, `'`, `''`).Replace(v) + `'`
	}
	return `'` + strings.ReplaceAll(v, `'`, `''`) + `'`
}

// execRoleConfig runs ALTER ROLE <login> SET <name> = '<value>' / RESET <name>.
// The role name is double-quoted (case preserved, special chars safe); the GUC name is a
// validated bare identifier (unquoted — GUC names are case-insensitive and may be namespaced);
// the value is a quoted literal.
func execRoleConfig(ctx context.Context, q Querier, operation string, args map[string]string) (sql string, msg string, err error) {
	login := strings.TrimSpace(args["loginname"])
	name := strings.TrimSpace(args["config_name"])
	if login == "" || strings.ContainsAny(login, ",\x00") {
		return "", "", fmt.Errorf("invalid role name: %q", login)
	}
	if !gucNameRE.MatchString(name) {
		return "", "", fmt.Errorf("invalid setting name: %q", name)
	}
	qlogin := `"` + strings.ReplaceAll(login, `"`, `""`) + `"`
	if operation == "reset_config" {
		sql = fmt.Sprintf("ALTER ROLE %s RESET %s", qlogin, name)
	} else {
		sql = fmt.Sprintf("ALTER ROLE %s SET %s = %s", qlogin, name, quoteConfigValue(args["config_value"]))
	}
	if _, err := q.Exec(ctx, sql); err != nil {
		return sql, "", err // return the SQL even on failure so it can be surfaced
	}
	return sql, "ok", nil
}

// ExecuteOperation runs one operation on q (a *pgx.Conn for autocommit, or a pgx.Tx to
// participate in a per-cluster transaction). It returns the executed SQL text (for display;
// function-mode bind params are inlined) — non-empty whenever a statement was actually sent,
// including on execution error — plus a status message.
func ExecuteOperation(ctx context.Context, q Querier, fn model.DBFunction, operation string, args map[string]string) (sql string, msg string, err error) {
	// Role GUC settings are written with hardcoded ALTER ROLE SET/RESET (no template).
	if operation == "set_config" || operation == "reset_config" {
		return execRoleConfig(ctx, q, operation, args)
	}

	call := strings.TrimSpace(fn.Call)
	if call == "" {
		return "", "", errCallNotConfigured()
	}

	execution := model.NormalizeExecution(fn.Execution)
	query, values, useQuery, err := calltemplate.Build(call, args, operation, execution)
	if err != nil {
		return "", "", err
	}

	// statement/block SQL is already fully embedded; function-mode uses pgx binds, so inline
	// the values into a readable, copy-pasteable approximation for display.
	sql = query
	if useQuery {
		sql = inlineParams(query, values)
		msg, err = runQuery(ctx, q, query, values...)
		return sql, msg, err
	}
	if _, err := q.Exec(ctx, query); err != nil {
		return sql, "", err
	}
	return sql, "ok", nil
}

// inlineParams substitutes $1,$2,… in a function-mode query with quoted literals, for a
// human-readable display string (NOT re-executed). String values become 'escaped'; []string
// (from ARRAY || concat) become ARRAY['a','b']::text[]. Highest index first so $10 isn't
// clobbered by the $1 replacement.
func inlineParams(query string, values []any) string {
	for i := len(values); i >= 1; i-- {
		token := fmt.Sprintf("$%d", i)
		query = strings.ReplaceAll(query, token, sqlLiteral(values[i-1]))
	}
	return query
}

func sqlLiteral(v any) string {
	switch t := v.(type) {
	case []string:
		// The query token already carries a ::text[] cast (e.g. $1::text[]), so don't add one.
		parts := make([]string, len(t))
		for i, s := range t {
			parts[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
		return "ARRAY[" + strings.Join(parts, ", ") + "]"
	case string:
		return "'" + strings.ReplaceAll(t, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprintf("%v", t), "'", "''") + "'"
	}
}

// CallFunction is an alias for ExecuteOperation (Wails-era name).
func CallFunction(ctx context.Context, conn *pgx.Conn, fn model.DBFunction, operation string, args map[string]string) (sql string, msg string, err error) {
	return ExecuteOperation(ctx, conn, fn, operation, args)
}

func runQuery(ctx context.Context, q Querier, query string, values ...any) (string, error) {
	rows, err := q.Query(ctx, query, values...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		return "ok", rows.Err()
	}

	var msg string
	if err := rows.Scan(&msg); err != nil {
		return "ok", nil
	}
	if msg == "" {
		return "ok", nil
	}
	return msg, rows.Err()
}

func errCallNotConfigured() error {
	return &callNotConfiguredError{}
}

type callNotConfiguredError struct{}

func (e *callNotConfiguredError) Error() string {
	return "database call template is not configured"
}
