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
