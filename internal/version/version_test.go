package version

import "testing"

func TestGet_defaults(t *testing.T) {
	i := Get()
	if i.Version == "" {
		t.Fatal("expected default version")
	}
}

func TestInfo_String(t *testing.T) {
	got := Info{Version: "1.2.3", Commit: "dev"}.String()
	if got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	got = Info{Version: "1.2.3", Commit: "abc1234"}.String()
	if got != "1.2.3 (abc1234)" {
		t.Fatalf("got %q", got)
	}
}
