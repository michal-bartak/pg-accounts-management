package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

// The two files carry disjoint halves of the config: cluster inventory never reaches config.yaml,
// and app configuration never reaches clusters.yaml.
func TestClusterDataStaysOutOfConfigYAML(t *testing.T) {
	s := tmpStore(t)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddCluster(model.ClusterInput{
		Alias: "c1", Host: "h1", Database: "db", Category: "production", ConnectUser: "u",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTargets(model.TargetSelection{CategoryIDs: []string{"production"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateSearchColumns([]model.SearchColumn{{Label: "Name", Template: "${{full_name}}"}}); err != nil {
		t.Fatal(err)
	}

	main := readFileString(t, s.path)
	for _, key := range []string{"categories:", "clusters:", "targets:"} {
		if strings.Contains(main, key) {
			t.Errorf("config.yaml contains %q; it moved to clusters.yaml\n%s", key, main)
		}
	}
	if !strings.Contains(main, "search_columns:") {
		t.Errorf("config.yaml is missing search_columns:\n%s", main)
	}

	clusters := readFileString(t, s.clustersPath())
	for _, key := range []string{"categories:", "clusters:", "targets:"} {
		if !strings.Contains(clusters, key) {
			t.Errorf("clusters.yaml is missing %q\n%s", key, clusters)
		}
	}
	if strings.Contains(clusters, "db_functions:") {
		t.Errorf("clusters.yaml contains db_functions:; it belongs in config.yaml\n%s", clusters)
	}
}

// clusters.yaml holds the optional per-cluster password, so it must be owner-only like config.yaml.
func TestClustersFileIsPrivate(t *testing.T) {
	s := tmpStore(t)
	if _, err := s.AddCluster(model.ClusterInput{
		Alias: "c1", Host: "h1", Database: "db", Category: "production", ConnectUser: "u",
		Password: "s3cret",
	}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(s.clustersPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("clusters.yaml mode = %o, want 600 (it holds the per-cluster password)", got)
	}
}

// A config.yaml with no clusters.yaml beside it is a fresh install: default groups, no clusters.
func TestLoadToleratesMissingClustersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path}
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if len(got.Categories) != len(DefaultConfig().Categories) {
		t.Fatalf("categories = %+v, want the built-in defaults", got.Categories)
	}
	if len(got.Clusters) != 0 {
		t.Fatalf("clusters = %+v, want none", got.Clusters)
	}
	if len(got.Targets.CategoryIDs) != 0 || len(got.Targets.ClusterIDs) != 0 {
		t.Fatalf("targets = %+v, want empty", got.Targets)
	}
}

// An explicitly empty cluster list must round-trip. There is no default cluster set to resurrect,
// and a user who deleted their last cluster must not find it back after a restart.
func TestEmptyClusterListRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, filepath.Join(dir, "clusters.yaml"), "version: 1\ncategories:\n  - id: prod\n    label: Prod\nclusters: []\n")

	s := &Store{path: path}
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Clusters; len(got) != 0 {
		t.Fatalf("clusters = %+v, want none", got)
	}
	if got := s.Get().Categories; len(got) != 1 || got[0].ID != "prod" {
		t.Fatalf("categories = %+v, want just prod (not the defaults)", got)
	}
}

// Once clusters.yaml exists it is the only source of cluster data; leftover keys in config.yaml
// are ignored. This pins the no-migration decision.
func TestClustersFileWinsOverLegacyKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, path, "version: 1\ncategories:\n  - id: stale\n    label: Stale\nclusters:\n  - id: old\n    alias: old\n    host: h\n    database: db\n    category: stale\n")
	writeFile(t, filepath.Join(dir, "clusters.yaml"), "version: 1\ncategories:\n  - id: live\n    label: Live\nclusters:\n  - id: new\n    alias: new\n    host: h\n    database: db\n    category: live\n")

	s := &Store{path: path}
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if len(got.Clusters) != 1 || got.Clusters[0].Alias != "new" {
		t.Fatalf("clusters = %+v, want only the clusters.yaml one", got.Clusters)
	}
	if len(got.Categories) != 1 || got.Categories[0].ID != "live" {
		t.Fatalf("categories = %+v, want only the clusters.yaml one", got.Categories)
	}
}

// First run: neither file exists. This is the path NewStore takes on a fresh install (it resolves
// the real config dir, so the sequence is reproduced here rather than calling it). Nothing may
// error, and both files must land on disk with the built-in defaults.
func TestFirstRunCreatesBothFiles(t *testing.T) {
	dir := t.TempDir()
	s := &Store{path: filepath.Join(dir, "config.yaml")}

	if err := s.Load(); err != nil {
		t.Fatalf("Load() on an empty directory: %v", err)
	}
	if err := s.writeMissingFiles(); err != nil {
		t.Fatalf("writeMissingFiles(): %v", err)
	}

	for _, name := range []string{"config.yaml", "clusters.yaml"} {
		fi, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s was not created: %v", name, err)
		}
		if got := fi.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %o, want 600", name, got)
		}
	}

	// The defaults are in memory and survive a reload from the files just written.
	reloaded := &Store{path: filepath.Join(dir, "config.yaml")}
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := reloaded.Get()
	if len(got.Categories) != len(DefaultConfig().Categories) {
		t.Errorf("categories = %+v, want the built-in defaults", got.Categories)
	}
	if len(got.Clusters) != 0 {
		t.Errorf("clusters = %+v, want none", got.Clusters)
	}
	if got.DBFunctions.CreateRole.Call != DefaultConfig().DBFunctions.CreateRole.Call {
		t.Errorf("create_role = %q, want the built-in default", got.DBFunctions.CreateRole.Call)
	}
	if len(got.CommentFields) == 0 || len(got.SearchColumns) == 0 {
		t.Errorf("comment fields / search columns lost their defaults: %+v / %+v", got.CommentFields, got.SearchColumns)
	}
}

// The real first-run path: NewStore resolves the config directory itself, creates it, and seeds
// both files. HOME/APPDATA are redirected so this exercises ConfigDir's actual per-OS branch
// (including its MkdirAll) rather than a hand-built path.
func TestNewStoreFirstRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home) // windows branch
	t.Setenv("XDG_CONFIG_HOME", "")

	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() on a first run: %v", err)
	}
	for _, p := range []string{s.ConfigPath(), s.ClustersPath()} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s was not created: %v", p, err)
		}
	}
	if got := s.Get(); len(got.Categories) == 0 || got.DBFunctions.CreateRole.Call == "" {
		t.Fatalf("first run produced an unseeded config: %+v", got)
	}
	// Second launch over the files just written must also be clean.
	if _, err := NewStore(); err != nil {
		t.Fatalf("NewStore() on the second run: %v", err)
	}
}

// A config.yaml left over from before the split still carries categories/clusters/targets. Those
// keys are simply ignored (yaml.Unmarshal has KnownFields off) and dropped by the next write —
// there is deliberately no migration, and loading such a file must not error.
func TestLegacyKeysInConfigAreIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, path, "version: 1\nclusters:\n  - id: old\n    alias: old\n    host: h\n    database: db\n    category: production\ntargets:\n  category_ids: [production]\n")

	s := &Store{path: path}
	if err := s.Load(); err != nil {
		t.Fatalf("Load() on a pre-split config: %v", err)
	}
	if got := s.Get().Clusters; len(got) != 0 {
		t.Fatalf("clusters = %+v, want none (the stale config.yaml key is ignored)", got)
	}
	if got := s.Get().Targets.CategoryIDs; len(got) != 0 {
		t.Fatalf("targets = %+v, want empty", got)
	}
}

// A clusters.yaml that exists but does not parse is a hard error, and must leave the already
// loaded config alone rather than swapping in an empty one.
func TestCorruptClustersFileDoesNotClobberState(t *testing.T) {
	s := tmpStore(t)
	if _, err := s.AddCluster(model.ClusterInput{
		Alias: "c1", Host: "h1", Database: "db", Category: "production", ConnectUser: "u",
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s.clustersPath(), "clusters: [ this is not: valid yaml\n")

	if err := s.Load(); err == nil {
		t.Fatal("Load() succeeded on a corrupt clusters.yaml; want an error")
	}
	if got := s.Get().Clusters; len(got) != 1 || got[0].Alias != "c1" {
		t.Fatalf("clusters = %+v, want the previously loaded set to survive", got)
	}
}

// A store with no path (NewStoreFromConfig) must not write anything. Without the guard, the
// derived clusters path would be a relative "clusters.yaml" in the working directory.
func TestPathlessStoreWritesNothing(t *testing.T) {
	s := NewStoreFromConfig(DefaultConfig())
	if err := s.UpdateTargets(model.TargetSelection{CategoryIDs: []string{"production"}}); err == nil {
		t.Fatal("UpdateTargets() on a path-less store succeeded; want an error")
	}
	if _, err := os.Stat("clusters.yaml"); !os.IsNotExist(err) {
		os.Remove("clusters.yaml")
		t.Fatal("a path-less store wrote clusters.yaml into the working directory")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
