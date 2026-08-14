package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.4.0", "0.3.0", 1},
		{"0.3.0", "0.4.0", -1},
		{"0.3.0", "0.3.0", 0},
		{"v0.4.0", "0.3.0", 1},   // leading v stripped
		{"0.3.0", "v0.3.0", 0},   // both forms equal
		{"1.0.0", "0.9.9", 1},    // major dominates
		{"0.3.10", "0.3.2", 1},   // numeric, not lexicographic
		{"0.4.0-rc1", "0.3.0", 1}, // prerelease suffix ignored, still newer
		{"0.3", "0.3.0", 0},      // missing patch = 0
		{"garbage", "0.3.0", -1}, // non-numeric sorts as 0.0.0
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestOwnerRepo(t *testing.T) {
	o, r, ok := ownerRepo("https://github.com/michal-bartak/pgcowboy")
	if !ok || o != "michal-bartak" || r != "pgcowboy" {
		t.Fatalf("ownerRepo = %q/%q ok=%v", o, r, ok)
	}
	if _, _, ok := ownerRepo("https://example.com/x/y"); ok {
		t.Fatal("non-github URL should not parse")
	}
}

// serve stands up a fake GitHub API returning the given status/body for releases/latest.
func serve(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/michal-bartak/pgcowboy/releases/latest" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	old := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = old })
}

const repo = "https://github.com/michal-bartak/pgcowboy"

func TestCheck_UpdateAvailable(t *testing.T) {
	serve(t, http.StatusOK, `{"tag_name":"v0.4.0","name":"0.4.0","html_url":"https://example/rel/0.4.0","body":"notes"}`)
	got, err := Check(context.Background(), "0.3.0", repo)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdateAvailable || got.LatestVersion != "0.4.0" || got.ReleaseURL == "" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheck_UpToDate(t *testing.T) {
	serve(t, http.StatusOK, `{"tag_name":"v0.3.0","html_url":"https://example/rel/0.3.0"}`)
	got, err := Check(context.Background(), "0.3.0", repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.UpdateAvailable {
		t.Fatalf("expected up to date, got %+v", got)
	}
}

func TestCheck_NoReleases404(t *testing.T) {
	serve(t, http.StatusNotFound, `{"message":"Not Found"}`)
	got, err := Check(context.Background(), "0.3.0", repo)
	if err != nil {
		t.Fatalf("404 should not be an error: %v", err)
	}
	if got.UpdateAvailable || got.LatestVersion != "" {
		t.Fatalf("404 should be up-to-date/empty, got %+v", got)
	}
}

func TestCheck_ServerError(t *testing.T) {
	serve(t, http.StatusInternalServerError, `oops`)
	if _, err := Check(context.Background(), "0.3.0", repo); err == nil {
		t.Fatal("HTTP 500 should error")
	}
}

func TestPending(t *testing.T) {
	// A seen version still newer than current → pending (badge lit across restart), no network.
	got := Pending("0.4.0", "0.3.0", repo)
	if !got.UpdateAvailable || got.LatestVersion != "0.4.0" ||
		got.ReleaseURL != repo+"/releases/latest" || got.CurrentVersion != "0.3.0" {
		t.Fatalf("expected pending 0.4.0, got %+v", got)
	}
	// Leading "v" and surrounding space are tolerated.
	if got := Pending(" v0.4.0 ", "0.3.0", repo); !got.UpdateAvailable || got.LatestVersion != "0.4.0" {
		t.Fatalf("expected v-prefixed pending, got %+v", got)
	}
	// Upgraded past (or equal to) the seen version → nothing pending.
	if got := Pending("0.4.0", "0.4.0", repo); got.UpdateAvailable {
		t.Fatalf("equal versions must not be pending: %+v", got)
	}
	if got := Pending("0.4.0", "0.5.0", repo); got.UpdateAvailable {
		t.Fatalf("current ahead of seen must not be pending: %+v", got)
	}
	// No seen version → nothing pending.
	if got := Pending("", "0.3.0", repo); got.UpdateAvailable {
		t.Fatalf("empty seen must not be pending: %+v", got)
	}
}
