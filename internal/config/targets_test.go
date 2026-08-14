package config

import (
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

func TestUpdateTargetsRoundTrip(t *testing.T) {
	s := tmpStore(t)
	want := model.TargetSelection{
		CategoryIDs: []string{"production", "uat"},
		ClusterIDs:  []string{"c-1"},
	}
	if err := s.UpdateTargets(want); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Targets; len(got.CategoryIDs) != 2 || len(got.ClusterIDs) != 1 {
		t.Fatalf("in-memory targets not set: %+v", got)
	}

	// Reload from disk into a fresh store: the selection must survive a restart.
	fresh := &Store{path: s.path}
	if err := fresh.Load(); err != nil {
		t.Fatal(err)
	}
	got := fresh.Get().Targets
	if len(got.CategoryIDs) != 2 || got.CategoryIDs[0] != "production" || got.CategoryIDs[1] != "uat" {
		t.Fatalf("category ids not persisted: %+v", got.CategoryIDs)
	}
	if len(got.ClusterIDs) != 1 || got.ClusterIDs[0] != "c-1" {
		t.Fatalf("cluster ids not persisted: %+v", got.ClusterIDs)
	}

	// Empty selection round-trips as empty (UI treats it as the "all groups" default).
	if err := fresh.UpdateTargets(model.TargetSelection{}); err != nil {
		t.Fatal(err)
	}
	reloaded := &Store{path: s.path}
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Get().Targets; len(got.CategoryIDs) != 0 || len(got.ClusterIDs) != 0 {
		t.Fatalf("empty targets should persist empty: %+v", got)
	}
}
