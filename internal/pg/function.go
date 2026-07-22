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

var (
	roleIdentRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	gucNameRE   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)
)

// Querier is the subset of pgx used to run an operation. Both *pgx.Conn and pgx.Tx satisfy it,
// so an operation can run either standalone (autocommit) or inside a per-cluster transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// execRoleConfig runs ALTER ROLE <login> SET <name> = '<value>' / RESET <name>.
// Role name and GUC name are validated as identifiers; the value is a quoted literal.
func execRoleConfig(ctx context.Context, q Querier, operation string, args map[string]string) (string, error) {
	login := strings.TrimSpace(args["loginname"])
	name := strings.TrimSpace(args["config_name"])
	if !roleIdentRE.MatchString(login) {
		return "", fmt.Errorf("invalid role name: %q", login)
	}
	if !gucNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid setting name: %q", name)
	}
	var sql string
	if operation == "reset_config" {
		sql = fmt.Sprintf("ALTER ROLE %s RESET %s", login, name)
	} else {
		value := strings.ReplaceAll(args["config_value"], "'", "''")
		sql = fmt.Sprintf("ALTER ROLE %s SET %s = '%s'", login, name, value)
	}
	if _, err := q.Exec(ctx, sql); err != nil {
		return "", err
	}
	return "ok", nil
}

// ExecuteOperation runs one operation on q (a *pgx.Conn for autocommit, or a pgx.Tx to
// participate in a per-cluster transaction).
func ExecuteOperation(ctx context.Context, q Querier, fn model.DBFunction, operation string, args map[string]string) (string, error) {
	// Role GUC settings are written with hardcoded ALTER ROLE SET/RESET (no template).
	if operation == "set_config" || operation == "reset_config" {
		return execRoleConfig(ctx, q, operation, args)
	}

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
		return runQuery(ctx, q, query, values...)
	}
	tag, err := q.Exec(ctx, query)
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
