package pg

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/michalbartak/dbaccounts/internal/calltemplate"
	"github.com/michalbartak/dbaccounts/internal/model"
)

func ExecuteOperation(ctx context.Context, conn *pgx.Conn, fn model.DBFunction, operation string, args map[string]string) (string, error) {
	call := strings.TrimSpace(fn.Call)
	if call == "" {
		return "", errCallNotConfigured()
	}

	execution := model.NormalizeExecution(fn.Execution)
	query, values, useQuery, err := calltemplate.Build(call, args, operation, execution)
	if err != nil {
		return "", err
	}

	if useQuery {
		return runQuery(ctx, conn, query, values...)
	}
	tag, err := conn.Exec(ctx, query)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() > 0 {
		return "ok", nil
	}
	return "ok", nil
}

// CallFunction is an alias for ExecuteOperation (Wails-era name).
func CallFunction(ctx context.Context, conn *pgx.Conn, fn model.DBFunction, operation string, args map[string]string) (string, error) {
	return ExecuteOperation(ctx, conn, fn, operation, args)
}

func runQuery(ctx context.Context, conn *pgx.Conn, query string, values ...any) (string, error) {
	rows, err := conn.Query(ctx, query, values...)
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
