package config

import "testing"

func TestValidateParentRoles(t *testing.T) {
	got, err := validateParentRoles([]string{" gr_a ", "gr_b", "gr_a", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "gr_a" || got[1] != "gr_b" {
		t.Fatalf("trim/dedupe failed: %v", got)
	}
	if _, err := validateParentRoles([]string{"bad-name"}); err == nil {
		t.Fatal("expected error for non-identifier parent group")
	}
}

func TestSanitizeParentRoles(t *testing.T) {
	got := sanitizeParentRoles([]string{" x ", "x", "", "y"})
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("trim/dedupe failed: %v", got)
	}
}
