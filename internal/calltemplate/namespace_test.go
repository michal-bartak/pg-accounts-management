package calltemplate

import (
	"strings"
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

// The bare ${…} namespace is a closed set of built-ins; ${{…}} is always a configured comment
// field. These tests pin the separation, and in particular the collision cases that had no
// coverage while the two shared one namespace.

func TestNamespaces_commentFieldKeyedLikeBuiltin_statement(t *testing.T) {
	// A deployment whose JSON comment carries its own "comment" key. Both meanings must be
	// reachable in the same template: the whole comment, and that one field.
	sql, _, _, err := Build(
		"COMMENT ON ROLE ${loginname} IS ${comment} -- ${{comment}}",
		map[string]string{
			"loginname":  "jdoe",
			"comment":    `{"comment":"hi"}`,
			"cf:comment": `"hi"`,
		},
		"set_comment", model.ExecutionStatement, "comment")
	if err != nil {
		t.Fatal(err)
	}
	want := `COMMENT ON ROLE "jdoe" IS '{"comment":"hi"}' -- 'hi'`
	if sql != want {
		t.Fatalf("got:  %s\nwant: %s", sql, want)
	}
}

// TestNamespaces_commentFieldKeyedLikeBuiltin_function is the regression test for the bug the
// namespace split fixes: buildFunctionQuery used to take the SQL shape from a built-ins-first
// lookup but the bound VALUE from comment-field-set membership. On a colliding name those
// disagreed, so ${comment} bound the whole comment through the comment-field JSON decoder —
// reordering keys, turning "" into NULL and a numeric-looking comment into a float.
func TestNamespaces_commentFieldKeyedLikeBuiltin_function(t *testing.T) {
	_, vals, _, err := Build(
		"admin.set(${loginname}, ${comment}, ${{comment}})",
		map[string]string{
			"loginname":  "jdoe",
			"comment":    `{"b":1,"a":2}`,
			"cf:comment": `"note"`,
		},
		"set_comment", model.ExecutionFunction, "comment")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 3 {
		t.Fatalf("want 3 binds, got %#v", vals)
	}
	// The built-in binds the comment text VERBATIM — not JSON-decoded and re-marshaled.
	if vals[1] != `{"b":1,"a":2}` {
		t.Fatalf("${comment} must bind the raw comment, got %#v", vals[1])
	}
	if vals[2] != "note" {
		t.Fatalf("${{comment}} must bind the field value, got %#v", vals[2])
	}
}

func TestNamespaces_builtinValueNotDecoded(t *testing.T) {
	// Same shape as above for the two other collidable built-ins, and the cases that used to be
	// mangled by the comment-field decoder: an empty comment and a numeric-looking one.
	for _, tc := range []struct {
		name    string
		comment string
	}{
		{"empty comment stays a string", ""},
		{"numeric comment stays a string", "42"},
		{"bool-looking comment stays a string", "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, vals, _, err := Build("admin.set(${loginname}, ${comment})",
				map[string]string{"loginname": "jdoe", "comment": tc.comment},
				"set_comment", model.ExecutionFunction, "comment")
			if err != nil {
				t.Fatal(err)
			}
			if vals[1] != tc.comment {
				t.Fatalf("${comment} bind = %#v, want the string %q", vals[1], tc.comment)
			}
		})
	}
}

func TestNamespaces_commentFieldKeyedLoginname(t *testing.T) {
	// ${loginname} stays the identifier; ${{loginname}} is the field's typed value.
	sql, _, _, err := Build("CREATE ROLE ${loginname} -- ${{loginname}}",
		map[string]string{"loginname": "jdoe", "cf:loginname": `"shadow"`},
		"create_role", model.ExecutionStatement, "loginname")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `CREATE ROLE "jdoe" -- 'shadow'` {
		t.Fatalf("got: %s", sql)
	}

	// Function mode: the identifier must not be run through the comment-field decoder (a
	// numeric-looking rolename would otherwise bind as a float).
	_, vals, _, err := Build("admin.create(${loginname}, ${{loginname}})",
		map[string]string{"loginname": "12345", "cf:loginname": `"shadow"`},
		"create_role", model.ExecutionFunction, "loginname")
	if err != nil {
		t.Fatal(err)
	}
	if vals[0] != "12345" {
		t.Fatalf("${loginname} bind = %#v, want the string \"12345\"", vals[0])
	}
	if vals[1] != "shadow" {
		t.Fatalf("${{loginname}} bind = %#v", vals[1])
	}
}

func TestNamespaces_commentFieldKeyedParentRoles(t *testing.T) {
	// ${parent_roles} keeps its list semantics in both modes; ${{parent_roles}} is a scalar field.
	sql, _, _, err := Build("GRANT ${parent_roles} TO ${loginname} -- ${{parent_roles}}",
		map[string]string{"loginname": "jdoe", "parent_roles": "a,b", "cf:parent_roles": `"x"`},
		"create_role", model.ExecutionStatement, "parent_roles")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `GRANT "a", "b" TO "jdoe" -- 'x'` {
		t.Fatalf("got: %s", sql)
	}

	q, vals, _, err := Build("admin.create(${loginname}, ${parent_roles}, ${{parent_roles}})",
		map[string]string{"loginname": "jdoe", "parent_roles": "a,b", "cf:parent_roles": `"x"`},
		"create_role", model.ExecutionFunction, "parent_roles")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "ARRAY['a', 'b']") {
		t.Fatalf("${parent_roles} should stay an inline array: %s", q)
	}
	if vals[len(vals)-1] != "x" {
		t.Fatalf("${{parent_roles}} bind = %#v", vals[len(vals)-1])
	}
}

func TestNamespaces_doubleFormRejections(t *testing.T) {
	cases := []struct {
		name, call, op string
		fields         []string
		wantSubstr     string
	}{
		{
			name: "unconfigured field", call: "CREATE ROLE ${loginname} -- ${{dept}}",
			op: "create_role", fields: []string{"full_name"}, wantSubstr: "${{dept}}",
		},
		{
			name: "op without comment fields", call: "ALTER ROLE ${loginname} PASSWORD ${new_password} -- ${{full_name}}",
			op: "change_password", fields: []string{"full_name"}, wantSubstr: "only available for create_role and set_comment",
		},
		{
			name: "non-identifier key", call: "CREATE ROLE ${loginname} -- ${{a-b}}",
			op: "create_role", fields: []string{"full_name"}, wantSubstr: "invalid comment-field placeholder",
		},
		{
			name: "empty double form", call: "CREATE ROLE ${loginname} -- ${{}}",
			op: "create_role", fields: []string{"full_name"}, wantSubstr: "empty placeholder",
		},
		{
			// The deprecated array_concat form belongs to the built-in namespace; in the
			// comment-field namespace it is just an invalid key. (Asserted on set_comment because
			// create_role rejects any array_concat outside function mode earlier than the parse.)
			name: "array_concat is built-in only", call: "admin.set(${{array_concat:comment,a}})",
			op: "set_comment", fields: []string{"full_name"}, wantSubstr: "invalid comment-field placeholder",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCallTemplateWithExecution(tc.call, tc.op, model.ExecutionStatement, tc.fields...)
			if err == nil {
				t.Fatalf("expected an error for %s", tc.call)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("error %q should mention %q", err, tc.wantSubstr)
			}
		})
	}
}

func TestNamespaces_malformedPlaceholders(t *testing.T) {
	// Only the two exact forms are placeholders; anything else containing "${" is malformed and
	// must be rejected rather than silently emitted or half-substituted.
	for _, call := range []string{
		"CREATE ROLE ${loginname} -- ${{full_name}",
		"CREATE ROLE ${loginname} -- ${full_name",
		"CREATE ROLE ${loginname} -- ${a{b}",
		"CREATE ROLE ${loginname} -- ${",
	} {
		if err := ValidateCallTemplateWithExecution(call, "create_role", model.ExecutionStatement, "full_name"); err == nil {
			t.Fatalf("expected %q to be rejected", call)
		}
	}
	// ${} is an empty name, not a malformed brace pair.
	err := ValidateCallTemplateWithExecution("CREATE ROLE ${loginname} -- ${}", "create_role", model.ExecutionStatement)
	if err == nil || !strings.Contains(err.Error(), "empty placeholder") {
		t.Fatalf("want an empty-placeholder error, got %v", err)
	}
}

// TestNamespaces_valueContainingPlaceholderSyntax pins the removal of the post-substitution
// "unresolved placeholders" guard: it inspected the OUTPUT, so a comment whose text contained
// "${" failed the build. The template scan now happens before substitution instead.
func TestNamespaces_valueContainingPlaceholderSyntax(t *testing.T) {
	sql, _, _, err := Build("COMMENT ON ROLE ${loginname} IS ${comment}",
		map[string]string{"loginname": "jdoe", "comment": "budget ${x} approved"},
		"set_comment", model.ExecutionStatement)
	if err != nil {
		t.Fatalf("a value containing ${ must not fail the build: %v", err)
	}
	if !strings.Contains(sql, "budget ${x} approved") {
		t.Fatalf("value should be embedded verbatim: %s", sql)
	}
}
