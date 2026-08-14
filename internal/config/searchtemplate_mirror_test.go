package config

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/calltemplate"
)

// A search-column template is rendered by the FRONTEND (it is display text over the role comment,
// never SQL), but validated HERE when the user saves. So the grammar and the closed bare-namespace
// necessarily exist in both languages — the one duplication in this area that cannot be deleted.
//
// These tests pin it. Without them the two copies drift silently in the worst possible way: the
// backend accepts a template the frontend renders differently, or rejects one it renders fine.

func readAppJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../frontend/app.js")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The JS regex literal body must be character-for-character the Go pattern. Both are written as
// raw/literal forms with no escaping differences, so a plain comparison is meaningful.
func TestSearchTokenRegexMatchesFrontend(t *testing.T) {
	src := readAppJS(t)
	m := regexp.MustCompile(`const SEARCH_TOKEN_RE = /(.+?)/g;`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("SEARCH_TOKEN_RE not found in frontend/app.js — this guard is blind, fix the pattern above")
	}
	if m[1] != calltemplate.TokenPattern {
		t.Errorf("placeholder grammar has drifted between Go and the frontend:\n  app.js:        %s\n  TokenPattern:  %s", m[1], calltemplate.TokenPattern)
	}
}

// The bare ${…} namespace a search column may use is closed, and both sides must agree on its
// members: a name the frontend renders but the backend rejects (or vice versa) is a broken save.
func TestSearchBuiltinsMatchFrontend(t *testing.T) {
	src := readAppJS(t)
	m := regexp.MustCompile(`const SEARCH_BUILTINS = new Set\(\[(.*?)\]\);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("SEARCH_BUILTINS not found in frontend/app.js — this guard is blind, fix the pattern above")
	}
	var js []string
	for _, raw := range regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(m[1], -1) {
		js = append(js, raw[1])
	}
	var go_ []string
	for name := range searchBuiltins {
		go_ = append(go_, name)
	}
	sort.Strings(js)
	sort.Strings(go_)
	if strings.Join(js, ",") != strings.Join(go_, ",") {
		t.Errorf("search-column built-ins differ:\n  app.js: %v\n  Go:     %v", js, go_)
	}
}

// The frontend mirrors checkSearchTemplate so a bad row is flagged live, with a row number the
// backend error cannot carry. The two must guide the user to the SAME fix — the whole point of
// the message is naming the ${{…}} form, and a divergence here is a UX bug nothing else catches.
func TestSearchTemplateErrorGuidanceMatches(t *testing.T) {
	src := readAppJS(t)
	fn := regexp.MustCompile(`(?s)function searchTemplateError\(tmpl\) \{.*?\n\}`).FindString(src)
	if fn == "" {
		t.Fatal("searchTemplateError not found in frontend/app.js — this guard is blind")
	}
	// Both sides reject the same two structural problems and steer to the same replacement form.
	for _, want := range []string{
		"Unfinished placeholder",
		"Empty placeholder",
		"is not supported",
	} {
		if !strings.Contains(fn, want) {
			t.Errorf("frontend searchTemplateError no longer mentions %q", want)
		}
	}

	// And the Go side must still produce those same three cases.
	cases := []struct{ tmpl, want string }{
		{"${comment", "unfinished placeholder"},
		{"${}", "empty placeholder"},
		{"${full_name}", "is not supported"},
	}
	for _, tc := range cases {
		err := checkSearchTemplate(tc.tmpl)
		if err == nil {
			t.Errorf("checkSearchTemplate(%q) = nil, want an error", tc.tmpl)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("checkSearchTemplate(%q) = %v, want it to mention %q", tc.tmpl, err, tc.want)
		}
	}
}
