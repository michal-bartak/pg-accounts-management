package pg

import "testing"

func TestParseFullName(t *testing.T) {
	cases := []struct {
		name    string
		comment string
		want    string
	}{
		{"json with full_name", `{"full_name":"Alice Example","email":"a@x.com"}`, "Alice Example"},
		{"json with spaces trimmed", `{"full_name":"  Bob  "}`, "Bob"},
		{"json without full_name", `{"email":"a@x.com"}`, ""},
		{"json full_name empty", `{"full_name":""}`, ""},
		{"json full_name not string", `{"full_name":42}`, ""},
		{"plain text comment", "just a person", ""},
		{"empty comment", "", ""},
		{"whitespace comment", "   ", ""},
		{"invalid json", `{"full_name":`, ""},
		{"leading whitespace json", `  {"full_name":"Carol"}`, "Carol"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseFullName(tc.comment); got != tc.want {
				t.Fatalf("ParseFullName(%q) = %q, want %q", tc.comment, got, tc.want)
			}
		})
	}
}

func TestLikePattern(t *testing.T) {
	cases := []struct {
		term string
		want string
	}{
		{"ali", "%ali%"},
		{" ali ", "%ali%"},
		{"50%", `%50\%%`},
		{"a_b", `%a\_b%`},
		{`a\b`, `%a\\b%`},
		{"", "%%"},
	}
	for _, tc := range cases {
		if got := likePattern(tc.term); got != tc.want {
			t.Fatalf("likePattern(%q) = %q, want %q", tc.term, got, tc.want)
		}
	}
}

func TestInlineRoleName(t *testing.T) {
	cases := []struct {
		name  string
		query string
		login string
		want  string
	}{
		{
			"named placeholder",
			"SELECT 1 FROM pg_shdepend WHERE rolname = ${rolename}",
			"bartakm",
			"SELECT 1 FROM pg_shdepend WHERE rolname = 'bartakm'",
		},
		{
			"legacy positional bind",
			"SELECT 1 WHERE rolname = $1",
			"alice",
			"SELECT 1 WHERE rolname = 'alice'",
		},
		{
			"quote in the role name is escaped",
			"SELECT 1 WHERE rolname = ${rolename}",
			"o'brien",
			"SELECT 1 WHERE rolname = 'o''brien'",
		},
		{
			"every occurrence replaced",
			"SELECT ${rolename} WHERE a = ${rolename}",
			"x",
			"SELECT 'x' WHERE a = 'x'",
		},
	}
	for _, tc := range cases {
		if got := inlineRoleName(tc.query, tc.login); got != tc.want {
			t.Fatalf("%s: inlineRoleName() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRoleDependencyQueries(t *testing.T) {
	// Display-only: one entry, bind inlined, never re-executed.
	got := RoleDependencyQueries("SELECT 1 WHERE rolname = ${rolename}", "dep_test")
	if len(got) != 1 {
		t.Fatalf("RoleDependencyQueries returned %d entries, want 1", len(got))
	}
	if want := "SELECT 1 WHERE rolname = 'dep_test'"; got[0] != want {
		t.Fatalf("RoleDependencyQueries() = %q, want %q", got[0], want)
	}
}
