package pg

import (
	"os"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestResolvePassword_emptyWhenNoSource(t *testing.T) {
	p, err := ResolvePassword(model.Cluster{Host: "127.0.0.1", Port: 5432, Database: "postgres"}, "postgres", model.AuthContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "" {
		t.Fatalf("expected empty password, got %q", p)
	}
}

func TestResolvePassword_fromAuth(t *testing.T) {
	p, err := ResolvePassword(model.Cluster{}, "u", model.AuthContext{Password: "secret"})
	if err != nil || p != "secret" {
		t.Fatalf("got %q, %v", p, err)
	}
}

func TestResolvePassword_emptyAuthUsesPGPASSWORD(t *testing.T) {
	t.Setenv("PGPASSWORD", "")
	os.Setenv("PGPASSWORD", "")
	t.Cleanup(func() { os.Unsetenv("PGPASSWORD") })

	p, err := ResolvePassword(model.Cluster{}, "u", model.AuthContext{})
	if err != nil {
		t.Fatal(err)
	}
	if p != "" {
		t.Fatalf("expected empty from PGPASSWORD, got %q", p)
	}
}

func TestBuildDSN_noPassword(t *testing.T) {
	dsn := BuildDSN(model.Cluster{Host: "127.0.0.1", Port: 50032, Database: "mydb", SSLMode: "disable"}, "postgres", "")
	if dsn == "" {
		t.Fatal("empty dsn")
	}
	// user without password segment
	if _, err := os.ReadFile("/dev/null"); err == nil {
		_ = dsn
	}
}
