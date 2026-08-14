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

func TestRepoAndDocsURLDerivation(t *testing.T) {
	cases := []struct{ repo, wantRepo, wantDocs string }{
		{
			"https://github.com/michal-bartak/pgcowboy",
			"https://github.com/michal-bartak/pgcowboy",
			"https://michal-bartak.github.io/pgcowboy/",
		},
		{"git@github.com:acme/widgets.git", "https://github.com/acme/widgets", "https://acme.github.io/widgets/"},
		{"https://github.com/o/r.git/", "https://github.com/o/r", "https://o.github.io/r/"},
		{"", "", ""},
	}
	for _, c := range cases {
		old := Repo
		Repo = c.repo
		got := Get()
		Repo = old
		if got.RepoURL != c.wantRepo {
			t.Errorf("Repo %q → RepoURL %q, want %q", c.repo, got.RepoURL, c.wantRepo)
		}
		if got.DocsURL != c.wantDocs {
			t.Errorf("Repo %q → DocsURL %q, want %q", c.repo, got.DocsURL, c.wantDocs)
		}
	}
}
