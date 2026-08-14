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

func TestRunRoleBatch_validation(t *testing.T) {
	r := NewRunner(testStore(t))
	// Empty batch → error.
	if _, err := r.RunRoleBatch(model.RoleBatchRequest{}, nil); err == nil {
		t.Fatal("expected error for empty batch")
	}
	// Unknown cluster → error.
	_, err := r.RunRoleBatch(model.RoleBatchRequest{
		Clusters: []model.ClusterOps{{
			ClusterID:  "does-not-exist",
			Operations: []model.OperationSpec{{Operation: commands.OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: "x"}}},
		}},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "cluster") {
		t.Fatalf("got: %v", err)
	}
	// A malformed operation is rejected before anything connects (was covered via the removed
	// single-op Run path).
	_, err = r.RunRoleBatch(model.RoleBatchRequest{
		Clusters: []model.ClusterOps{{
			ClusterID:  "c-uat-1",
			Operations: []model.OperationSpec{{Operation: commands.OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: ""}}},
		}},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("got: %v", err)
	}
}

func TestRunRoleBatch_requiresProductionConfirm(t *testing.T) {
	r := NewRunner(testStore(t))
	_, err := r.RunRoleBatch(model.RoleBatchRequest{
		Clusters: []model.ClusterOps{{
			ClusterID:  "c-prod-1",
			Operations: []model.OperationSpec{{Operation: commands.OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: "jdoe"}}},
		}},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("got: %v", err)
	}
}

func TestRunRoleBatch_perClusterResults(t *testing.T) {
	// Validation + resolution + production gate pass; the transaction fails on connect (no server),
	// so each cluster gets one error result (not one per op).
	r := NewRunner(testStore(t))
	results, err := r.RunRoleBatch(model.RoleBatchRequest{
		ConfirmProduction: true,
		Auth:              model.AuthContext{User: "nobody", Password: "nopass"},
		Clusters: []model.ClusterOps{
			{ClusterID: "c-uat-1", Operations: []model.OperationSpec{
				{Operation: commands.OpGrantParents, GrantParents: &model.GrantParentsParams{LoginName: "u", ParentRoles: "gr_a"}},
				{Operation: commands.OpSetComment, SetComment: &model.SetCommentParams{LoginName: "u", Comment: "{}"}},
			}},
			{ClusterID: "c-uat-2", Operations: []model.OperationSpec{
				{Operation: commands.OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: "u"}},
			}},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 { // one row per cluster, not per op
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, res := range results {
		if res.Status != "error" || res.Alias == "" {
			t.Fatalf("expected connect error with alias, got %s alias=%q msg=%q", res.Status, res.Alias, res.Message)
		}
	}
}
