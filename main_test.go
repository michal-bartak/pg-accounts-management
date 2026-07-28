package main

import (
	"os"
	"strings"
	"testing"
)

// The app version is the embedded VERSION file (see main.go). Guard that the embed is wired
// and matches the repo-root VERSION, so a VERSION bump reflects in the app without ldflags.
func TestEmbeddedVersionMatchesFile(t *testing.T) {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(raw))
	if want == "" {
		t.Fatal("VERSION file is empty")
	}
	if got := strings.TrimSpace(versionFile); got != want {
		t.Fatalf("embedded VERSION = %q, want %q", got, want)
	}
}
