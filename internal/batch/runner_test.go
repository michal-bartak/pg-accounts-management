package batch

import (
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/commands"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/model"
)

func testStore(t *testing.T) *config.Store {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Clusters = []model.Cluster{
		{ID: "c-uat-1", Alias: "UAT1", Host: "127.0.0.1", Port: 5432, Database: "db", Category: "uat"},
		{ID: "c-uat-2", Alias: "UAT2", Host: "127.0.0.1", Port: 5433, Database: "db", Category: "uat"},
		{ID: "c-prod-1", Alias: "PROD1", Host: "10.0.0.1", Port: 5432, Database: "db", Category: "production"},
	}
	return config.NewStoreFromConfig(cfg)
}

func TestResolveClusters_byCategory(t *testing.T) {
	r := NewRunner(testStore(t))
	clusters, err := r.ResolveClusters(model.RunRequest{CategoryIDs: []string{"uat"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters", len(clusters))
	}
}

func TestResolveClusters_dedupeCategoryAndID(t *testing.T) {
	r := NewRunner(testStore(t))
	clusters, err := r.ResolveClusters(model.RunRequest{
		CategoryIDs: []string{"uat"},
		ClusterIDs:  []string{"c-uat-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters", len(clusters))
	}
}

func TestResolveClusters_noneMatched(t *testing.T) {
	r := NewRunner(testStore(t))
	_, err := r.ResolveClusters(model.RunRequest{CategoryIDs: []string{"nonexistent"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_removeRole_validatesRequest(t *testing.T) {
	r := NewRunner(testStore(t))
	_, err := r.Run(model.RunRequest{
		Operation:   commands.OpRemoveRole,
		CategoryIDs: []string{"uat"},
		RemoveRole:  &model.RemoveRoleParams{LoginName: ""},
	})
	if err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("got: %v", err)
	}
}

func TestRun_removeRole_requiresProductionConfirm(t *testing.T) {
	r := NewRunner(testStore(t))
	_, err := r.Run(model.RunRequest{
		Operation:           commands.OpRemoveRole,
		CategoryIDs:         []string{"production"},
		RemoveRole:          &model.RemoveRoleParams{LoginName: "jdoe"},
		ConfirmProduction:   false,
		Auth:                model.AuthContext{User: "postgres"},
	})
	if err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("got: %v", err)
	}
}

func TestRun_removeRole_buildsWithoutDB(t *testing.T) {
	// Exercises validation + resolution; runOne fails on connect (no server) — results still returned.
	r := NewRunner(testStore(t))
	results, err := r.Run(model.RunRequest{
		Operation:           commands.OpRemoveRole,
		CategoryIDs:         []string{"uat"},
		RemoveRole:          &model.RemoveRoleParams{LoginName: "jdoe"},
		ConfirmProduction:   true,
		Auth:                model.AuthContext{User: "nobody", Password: "nopass"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results", len(results))
	}
	for _, res := range results {
		if res.Status != "error" {
			t.Fatalf("expected connect error, got %s: %s", res.Status, res.Message)
		}
		if res.Alias == "" {
			t.Fatal("missing alias in result")
		}
	}
}

func TestRun_grantParents_and_changePassword_resolve(t *testing.T) {
	r := NewRunner(testStore(t))

	for _, op := range []struct {
		op  string
		req model.RunRequest
	}{
		{
			op: commands.OpGrantParents,
			req: model.RunRequest{
				Operation:    commands.OpGrantParents,
				CategoryIDs:  []string{"uat"},
				GrantParents: &model.GrantParentsParams{LoginName: "u", ParentRoles: "gr_a"},
				Auth:         model.AuthContext{User: "x"},
			},
		},
		{
			op: commands.OpRevokeParents,
			req: model.RunRequest{
				Operation:     commands.OpRevokeParents,
				CategoryIDs:   []string{"uat"},
				RevokeParents: &model.RevokeParentsParams{LoginName: "u", ParentRoles: "gr_a"},
				Auth:          model.AuthContext{User: "x"},
			},
		},
		{
			op: commands.OpChangePassword,
			req: model.RunRequest{
				Operation:      commands.OpChangePassword,
				CategoryIDs:    []string{"uat"},
				ChangePassword: &model.ChangePasswordParams{LoginName: "u", NewPassword: "p"},
				Auth:           model.AuthContext{User: "x"},
			},
		},
	} {
		t.Run(op.op, func(t *testing.T) {
			results, err := r.Run(op.req)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 2 {
				t.Fatalf("got %d results", len(results))
			}
		})
	}
}
