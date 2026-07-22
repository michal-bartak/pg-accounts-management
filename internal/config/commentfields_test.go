package config

import (
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestValidateCommentFields(t *testing.T) {
	got, err := validateCommentFields([]model.CommentField{
		{Key: " full_name ", Label: " Full name "},
		{Key: "e_mail", Label: ""},        // blank label defaults to key
		{Key: "full_name", Label: "Dupe"}, // dropped (dedupe by key)
		{Key: "  ", Label: "empty key"},   // dropped
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 fields, got %d: %v", len(got), got)
	}
	if got[0].Key != "full_name" || got[0].Label != "Full name" {
		t.Fatalf("trim failed: %+v", got[0])
	}
	if got[1].Key != "e_mail" || got[1].Label != "e_mail" {
		t.Fatalf("blank label should default to key: %+v", got[1])
	}
	if _, err := validateCommentFields([]model.CommentField{{Key: "bad-key", Label: "x"}}); err == nil {
		t.Fatal("expected error for non-identifier key")
	}
}

func TestSanitizeCommentFields(t *testing.T) {
	got := sanitizeCommentFields([]model.CommentField{
		{Key: " a ", Label: "A"},
		{Key: "a", Label: "dupe"},    // dropped
		{Key: "", Label: "x"},        // dropped
		{Key: "bad-key", Label: "y"}, // dropped (invalid, non-failing)
		{Key: "b", Label: ""},        // label defaults to key
	})
	if len(got) != 2 {
		t.Fatalf("want 2 fields, got %d: %v", len(got), got)
	}
	if got[0].Key != "a" || got[0].Label != "A" {
		t.Fatalf("unexpected first field: %+v", got[0])
	}
	if got[1].Key != "b" || got[1].Label != "b" {
		t.Fatalf("blank label should default to key: %+v", got[1])
	}
}

func TestDefaultCommentFields(t *testing.T) {
	got := defaultCommentFields()
	if len(got) != 2 || got[0].Key != "full_name" || got[1].Key != "e_mail" {
		t.Fatalf("unexpected defaults: %v", got)
	}
}

func TestUpdateCommentFieldsPersistsAndValidates(t *testing.T) {
	s := tmpStore(t)
	if err := s.UpdateCommentFields([]model.CommentField{
		{Key: "full_name", Label: "Full name"},
		{Key: "department", Label: "Department"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().CommentFields; len(got) != 2 || got[1].Key != "department" {
		t.Fatalf("not persisted: %v", got)
	}
	// Reload from disk to confirm it round-trips through YAML.
	s2 := &Store{path: s.path}
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().CommentFields; len(got) != 2 || got[1].Label != "Department" {
		t.Fatalf("did not round-trip: %v", got)
	}
	if err := s.UpdateCommentFields([]model.CommentField{{Key: "bad-key"}}); err == nil {
		t.Fatal("expected error for non-identifier key")
	}
}

func TestLoadInjectsDefaultCommentFields(t *testing.T) {
	s := tmpStore(t)
	s.cfg.CommentFields = nil
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{path: s.path}
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().CommentFields; len(got) != 2 || got[0].Key != "full_name" {
		t.Fatalf("Load should inject defaults when comment_fields omitted: %v", got)
	}
}
