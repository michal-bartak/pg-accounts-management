package pg

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/michalbartak/dbaccounts/internal/calltemplate"
	"github.com/michalbartak/dbaccounts/internal/model"
)

// fakeQuerier records Exec'd SQL; Query is only reached by function-mode ops (not tested here).
type fakeQuerier struct{ exec []string }

func (f *fakeQuerier) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.exec = append(f.exec, sql)
	return pgconn.CommandTag{}, nil
}

func (f *fakeQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, fmt.Errorf("Query not expected in this test")
}

func TestExecuteOperationReturnsStatementSQL(t *testing.T) {
	q := &fakeQuerier{}
	fn := model.DBFunction{Call: "COMMENT ON ROLE ${loginname} IS ${comment}", Execution: "statement"}
	sql, _, err := ExecuteOperation(context.Background(), q, fn, "set_comment", map[string]string{
		"loginname": "Mixed Case",
		"comment":   "hi 'there'",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `COMMENT ON ROLE "Mixed Case" IS 'hi ''there'''`
	if sql != want {
		t.Fatalf("statement sql = %q, want %q", sql, want)
	}
	if len(q.exec) != 1 || q.exec[0] != want {
		t.Fatalf("executed sql = %v, want [%q]", q.exec, want)
	}
}

func TestExecuteOperationReturnsConfigSQL(t *testing.T) {
	q := &fakeQuerier{}
	sql, _, err := ExecuteOperation(context.Background(), q, model.DBFunction{}, "set_config", map[string]string{
		"loginname":    "x",
		"config_name":  "search_path",
		"config_value": "public",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := `ALTER ROLE "x" SET search_path = 'public'`; sql != want {
		t.Fatalf("set_config sql = %q, want %q", sql, want)
	}

	q2 := &fakeQuerier{}
	sql, _, err = ExecuteOperation(context.Background(), q2, model.DBFunction{}, "reset_config", map[string]string{
		"loginname":   "x",
		"config_name": "search_path",
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := `ALTER ROLE "x" RESET search_path`; sql != want {
		t.Fatalf("reset_config sql = %q, want %q", sql, want)
	}
}

func TestInlineParams(t *testing.T) {
	// Simple string params (highest index replaced first so $10 isn't hit by $1).
	got := inlineParams("SELECT f($1, $2)", []any{"a", "b'c"})
	if want := "SELECT f('a', 'b''c')"; got != want {
		t.Fatalf("inlineParams = %q, want %q", got, want)
	}
	// Array-concat param already carries a ::text[] cast in the query token.
	got = inlineParams("SELECT g($1::text[])", []any{[]string{"gr_a", "gr_b"}})
	if want := "SELECT g(ARRAY['gr_a', 'gr_b']::text[])"; got != want {
		t.Fatalf("inlineParams array = %q, want %q", got, want)
	}
}

func TestFunctionModeSQLInlined(t *testing.T) {
	// The display SQL ExecuteOperation would produce for a function-mode op equals
	// inlineParams(Build(...)). Verify end-to-end via the same building blocks.
	query, values, useQuery, err := calltemplate.Build(
		"grant_parents(${loginname}, ${parent_roles})",
		map[string]string{"loginname": "x", "parent_roles": "gr_a,gr_b"},
		"grant_parents", "function",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !useQuery {
		t.Fatal("expected function mode (useQuery)")
	}
	got := inlineParams(query, values)
	if want := `SELECT grant_parents('x', 'gr_a,gr_b')`; got != want {
		t.Fatalf("function sql = %q, want %q", got, want)
	}
}
