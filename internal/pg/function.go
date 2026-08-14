package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/michal-bartak/pgcowboy/internal/calltemplate"
	"github.com/michal-bartak/pgcowboy/internal/model"
)

// Querier is the subset of pgx used to run an operation. Both *pgx.Conn and pgx.Tx satisfy it,
// so an operation can run either standalone (autocommit) or inside a per-cluster transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// ExecuteOperation runs one operation on q (a *pgx.Conn for autocommit, or a pgx.Tx to
// participate in a per-cluster transaction). It returns the executed SQL text (for display;
// function-mode bind params are inlined) — non-empty whenever a statement was actually sent,
// including on execution error — plus a status message.
func ExecuteOperation(ctx context.Context, q Querier, fn model.DBFunction, operation string, args map[string]string, commentFields ...string) (sql string, msg string, err error) {
	call := strings.TrimSpace(fn.Call)
	if call == "" {
		return "", "", errCallNotConfigured()
	}

	execution := model.NormalizeExecution(fn.Execution)
	query, values, useQuery, err := calltemplate.Build(call, args, operation, execution, commentFields...)
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
	case nil:
		// A nil bind (e.g. an empty/absent comment field) is a real SQL NULL, not the string
		// "<nil>". Render it unquoted so the display SQL matches what actually executes.
		return "NULL"
	case bool:
		if t {
			return "TRUE"
		}
		return "FALSE"
	case float64:
		return fmt.Sprintf("%v", t) // bare numeric literal (comment fields bind numbers as float64)
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
