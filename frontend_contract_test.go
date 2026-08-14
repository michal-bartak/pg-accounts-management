package main

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/calltemplate"
	"github.com/michal-bartak/pgcowboy/internal/model"
)

// The Settings editor lists the operations from two tables in frontend/app.js (DB_FUNCTIONS,
// DB_READS). Each entry names an operation `key` and the `prop` under which that operation's
// template lives in the JSON the backend sends. Neither is checked by the compiler, so adding an
// operation on one side only used to fail silently: a missing row means the template is simply
// not editable, and a stale `prop` means the editor reads/writes a field nobody serves.
//
// The default SQL itself is NOT duplicated any more — the frontend fetches it via
// GetDefaultTemplates — so these tests cover what is left of the contract.

var (
	fnEntryRE   = regexp.MustCompile(`\{\s*key:\s*'([a-z_]+)',\s*title:\s*'[^']*',\s*prop:\s*'([A-Za-z]+)'`)
	blockFnRE   = regexp.MustCompile(`(?s)const DB_FUNCTIONS = \[(.*?)\n\];`)
	blockReadRE = regexp.MustCompile(`(?s)const DB_READS = \[(.*?)\n\];`)
)

// parseTable pulls the (key, prop) pairs out of one app.js table.
func parseTable(t *testing.T, block *regexp.Regexp, src string, name string) map[string]string {
	t.Helper()
	m := block.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("could not locate %s in frontend/app.js — did the table's shape change?", name)
	}
	out := map[string]string{}
	for _, e := range fnEntryRE.FindAllStringSubmatch(m[1], -1) {
		out[e[1]] = e[2]
	}
	if len(out) == 0 {
		t.Fatalf("parsed no entries from %s — the entry shape changed, so this guard is blind", name)
	}
	return out
}

func readAppJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("frontend/app.js")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// jsonFieldNames returns the json tag names of a struct type, in declaration order.
func jsonFieldNames(v any) []string {
	rt := reflect.TypeOf(v)
	out := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				tag = tag[:j]
				break
			}
		}
		if tag != "" && tag != "-" {
			out = append(out, tag)
		}
	}
	return out
}

func TestFrontendDBFunctionsMatchBackend(t *testing.T) {
	table := parseTable(t, blockFnRE, readAppJS(t), "DB_FUNCTIONS")

	// Every listed key must be an operation the template engine actually knows.
	for key := range table {
		if calltemplate.AllowedPlaceholders(key) == nil {
			t.Errorf("app.js DB_FUNCTIONS lists %q, which calltemplate does not recognise as an operation", key)
		}
	}

	// Every listed prop must be a real field of model.DBFunctions, and every field must be listed
	// (an unlisted operation is not editable in Settings).
	want := jsonFieldNames(model.DBFunctions{})
	got := make([]string, 0, len(table))
	for _, prop := range table {
		got = append(got, prop)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("DB_FUNCTIONS props mismatch:\n  app.js:          %v\n  model.DBFunctions: %v", got, want)
	}
}

func TestFrontendDBReadsMatchBackend(t *testing.T) {
	table := parseTable(t, blockReadRE, readAppJS(t), "DB_READS")

	want := jsonFieldNames(model.DBReads{})
	got := make([]string, 0, len(table))
	for _, prop := range table {
		got = append(got, prop)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("DB_READS props mismatch:\n  app.js:       %v\n  model.DBReads: %v", got, want)
	}
}

// GetDefaultTemplates is what the editor's "Default" button reverts to, so it must actually carry
// every template — an empty one would silently blank a user's template on a button press.
func TestGetDefaultTemplatesIsComplete(t *testing.T) {
	app := &App{}
	def := app.GetDefaultTemplates()

	// Round-trip through JSON: that is the shape the frontend indexes by `prop`.
	var fns, reads map[string]map[string]any
	raw, err := json.Marshal(def.DBFunctions)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &fns); err != nil {
		t.Fatal(err)
	}
	if raw, err = json.Marshal(def.DBReads); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &reads); err != nil {
		t.Fatal(err)
	}

	for _, prop := range jsonFieldNames(model.DBFunctions{}) {
		if call, _ := fns[prop]["call"].(string); call == "" {
			t.Errorf("default db_function %s has an empty call template", prop)
		}
	}
	for _, prop := range jsonFieldNames(model.DBReads{}) {
		if q, _ := reads[prop]["query"].(string); q == "" {
			t.Errorf("default db_read %s has an empty query", prop)
		}
	}
}
