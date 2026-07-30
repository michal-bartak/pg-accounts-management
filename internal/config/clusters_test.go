package config

import (
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestSaveClustersAndCategories(t *testing.T) {
	s := tmpStore(t)

	// A new cluster (no id) + a new category (no id) round-trips: category id is slugified,
	// cluster id is a minted UUID, defaults are applied.
	err := s.SaveClustersAndCategories(
		[]model.Cluster{{Alias: "db1", Host: "h1", Database: "app", Category: "pre_prod"}},
		[]model.Category{{Label: "Pre Prod", Color: "#123456"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	cfg := s.Get()
	if len(cfg.Categories) != 1 || cfg.Categories[0].ID != "pre_prod" {
		t.Fatalf("category not slugified: %+v", cfg.Categories)
	}
	if len(cfg.Clusters) != 1 {
		t.Fatalf("want 1 cluster, got %d", len(cfg.Clusters))
	}
	c := cfg.Clusters[0]
	if c.ID == "" || len(c.ID) < 10 {
		t.Fatalf("cluster id should be a minted UUID: %q", c.ID)
	}
	if c.Port != 5432 || c.SSLMode != "prefer" {
		t.Fatalf("defaults not applied: port=%d sslmode=%q", c.Port, c.SSLMode)
	}

	// Editing preserves ids: pass the existing ids back with changed fields.
	existingID := c.ID
	err = s.SaveClustersAndCategories(
		[]model.Cluster{{ID: existingID, Alias: "db1-renamed", Host: "h1", Database: "app", Category: "pre_prod"}},
		[]model.Category{{ID: "pre_prod", Label: "Pre Prod"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Clusters[0]; got.ID != existingID || got.Alias != "db1-renamed" {
		t.Fatalf("edit should preserve id: %+v", got)
	}

	// Referential integrity: a cluster referencing a missing category is rejected.
	if err := s.SaveClustersAndCategories(
		[]model.Cluster{{Alias: "x", Host: "h", Database: "d", Category: "ghost"}},
		[]model.Category{{Label: "Pre Prod"}},
	); err == nil {
		t.Fatal("expected error for cluster referencing unknown group")
	}

	// Duplicate category ids (two labels slugging to the same id) rejected.
	if err := s.SaveClustersAndCategories(
		nil,
		[]model.Category{{Label: "Pre Prod"}, {Label: "pre  prod"}},
	); err == nil {
		t.Fatal("expected error for duplicate category id")
	}

	// Empty category set rejected.
	if err := s.SaveClustersAndCategories(nil, nil); err == nil {
		t.Fatal("expected error for empty category set")
	}

	// Missing required cluster field (host) rejected.
	if err := s.SaveClustersAndCategories(
		[]model.Cluster{{Alias: "x", Database: "d", Category: "pre_prod"}},
		[]model.Category{{Label: "Pre Prod"}},
	); err == nil {
		t.Fatal("expected error for missing host")
	}
}

// An optional per-cluster password round-trips through save + a fresh Load from disk.
func TestSaveClustersAndCategories_passwordRoundTrips(t *testing.T) {
	s := tmpStore(t)
	if err := s.SaveClustersAndCategories(
		[]model.Cluster{{Alias: "db1", Host: "h1", Database: "app", Category: "pre_prod", ConnectUser: "admin", Password: "s3cret"}},
		[]model.Category{{Label: "Pre Prod"}},
	); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Clusters[0].Password; got != "s3cret" {
		t.Fatalf("in-memory password not kept: %q", got)
	}

	// Reload from the same file to prove it was written to (and parsed from) YAML.
	reloaded := &Store{path: s.path}
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	c := reloaded.Get().Clusters[0]
	if c.Password != "s3cret" || c.ConnectUser != "admin" {
		t.Fatalf("password/connect_user did not survive disk round-trip: %+v", c)
	}
}
