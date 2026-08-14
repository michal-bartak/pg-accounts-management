package config

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

// Both savers write atomically (temp file + rename) and leave no stray temp files behind, and the
// persisted content round-trips through Load.
func TestSaveIsAtomicAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s := &Store{path: path, cfg: DefaultConfig()}

	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	// AddCluster writes clusters.yaml only; Save above wrote both.
	if _, err := s.AddCluster(model.ClusterInput{
		Alias: "c1", Host: "h1", Database: "db", Category: "production", ConnectUser: "u",
	}); err != nil {
		t.Fatal(err)
	}

	// Both real files exist (renames completed) and no leftover *.tmp siblings remain.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	// os.ReadDir sorts by filename, so clusters.yaml comes first.
	want := []string{"clusters.yaml", "config.yaml"}
	if !slices.Equal(names, want) {
		t.Fatalf("dir entries = %v, want %v (no temp files left behind)", names, want)
	}

	// Content round-trips: a fresh store loading the same path sees the added cluster.
	reloaded := &Store{path: path}
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	got := reloaded.Get().Clusters
	if len(got) != 1 || got[0].Alias != "c1" {
		t.Fatalf("reloaded clusters = %+v, want one cluster aliased c1", got)
	}
}

// Get returns a snapshot whose slices are independent of later in-place mutations of the
// store, so a caller iterating the result can't observe a concurrent write mid-iteration.
func TestGetReturnsIndependentCopy(t *testing.T) {
	s := tmpStore(t)
	c, err := s.AddCluster(model.ClusterInput{
		Alias: "orig", Host: "h", Database: "db", Category: "production", ConnectUser: "u",
	})
	if err != nil {
		t.Fatal(err)
	}
	snap := s.Get()
	if _, err := s.UpdateCluster(c.ID, model.ClusterInput{
		Alias: "changed", Host: "h", Database: "db", Category: "production", ConnectUser: "u",
	}); err != nil {
		t.Fatal(err)
	}
	if snap.Clusters[0].Alias != "orig" {
		t.Fatalf("snapshot mutated by later UpdateCluster: alias = %q, want orig", snap.Clusters[0].Alias)
	}
}

// Concurrent reads and writes must be race-free. Run with `go test -race` for this to have
// teeth; without -race it still exercises the locking for deadlocks.
func TestStoreConcurrentAccess(t *testing.T) {
	s := tmpStore(t)
	c, err := s.AddCluster(model.ClusterInput{
		Alias: "c", Host: "h", Database: "db", Category: "production", ConnectUser: "u",
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	const iters = 100

	// Writers: update the cluster and toggle target selection.
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			_, _ = s.UpdateCluster(c.ID, model.ClusterInput{
				Alias: "c", Host: "h", Database: "db", Category: "production", ConnectUser: "u",
			})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			_ = s.UpdateTargets(model.TargetSelection{CategoryIDs: []string{"production"}})
		}
	}()

	// Readers: Get / ClusterByID / ClustersByCategories.
	wg.Add(3)
	for r := 0; r < 3; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				cfg := s.Get()
				for _, cl := range cfg.Clusters {
					_ = cl.Alias
				}
				_, _ = s.ClusterByID(c.ID)
				_ = s.ClustersByCategories([]string{"production"})
			}
		}()
	}
	wg.Wait()
}
