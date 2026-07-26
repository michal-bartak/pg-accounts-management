package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/batch"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/envimport"
	"github.com/michalbartak/dbaccounts/internal/model"
	"github.com/michalbartak/dbaccounts/internal/pg"
	"github.com/michalbartak/dbaccounts/internal/version"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	store *config.Store
	batch *batch.Runner
}

func NewApp() (*App, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	return &App{
		store: store,
		batch: batch.NewRunner(store),
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetConfig() model.Config {
	return a.store.Get()
}

func (a *App) GetConfigPath() string {
	return a.store.ConfigPath()
}

// GetAppVersion returns build metadata (version, git commit, build date).
func (a *App) GetAppVersion() model.AppVersion {
	i := version.Get()
	return model.AppVersion{
		Version:   i.Version,
		Commit:    i.Commit,
		BuildDate: i.BuildDate,
	}
}

func (a *App) ReloadConfig() (model.Config, error) {
	if err := a.store.Load(); err != nil {
		return model.Config{}, err
	}
	return a.store.Get(), nil
}

func (a *App) SaveDBFunctions(fn model.DBFunctions) error {
	return a.store.UpdateDBFunctions(fn)
}

func (a *App) SaveDBReads(reads model.DBReads) error {
	return a.store.UpdateDBReads(reads)
}

func (a *App) SaveBatchSettings(batch model.BatchSettings) error {
	return a.store.UpdateBatch(batch)
}

func (a *App) SaveUISettings(ui model.UISettings) error {
	return a.store.UpdateUI(ui)
}

func (a *App) SaveParentRoles(roles []string) error {
	return a.store.UpdateParentRoles(roles)
}

func (a *App) SaveCommentFields(fields []model.CommentField) error {
	return a.store.UpdateCommentFields(fields)
}

// SaveTargetSelection persists the Operations-page target selection so it survives
// re-renders (e.g. after saving Settings) and app restarts.
func (a *App) SaveTargetSelection(t model.TargetSelection) error {
	return a.store.UpdateTargets(t)
}

func (a *App) AddCluster(in model.ClusterInput) (model.Cluster, error) {
	return a.store.AddCluster(in)
}

func (a *App) UpdateCluster(id string, in model.ClusterInput) (model.Cluster, error) {
	return a.store.UpdateCluster(id, in)
}

func (a *App) DeleteCluster(id string) error {
	return a.store.DeleteCluster(id)
}

func (a *App) AddCategory(in model.CategoryInput) (model.Category, error) {
	return a.store.AddCategory(in)
}

func (a *App) UpdateCategory(id string, in model.CategoryInput) (model.Category, error) {
	return a.store.UpdateCategory(id, in)
}

func (a *App) DeleteCategory(id string) error {
	return a.store.DeleteCategory(id)
}

// SaveClusters replaces the whole clusters+categories set at once (the staged Clusters editor's
// Save). Validates the full set atomically; nothing persists on error.
func (a *App) SaveClusters(cfg model.ClustersConfig) error {
	return a.store.SaveClustersAndCategories(cfg.Clusters, cfg.Categories)
}

func (a *App) ImportFromEnvironment() model.EnvImport {
	return envimport.FromEnvironment()
}

func (a *App) TestConnection(req model.TestConnectionRequest) error {
	cluster, ok := a.store.ClusterByID(req.ClusterID)
	if !ok {
		return fmt.Errorf("cluster not found")
	}
	return pg.TestConnection(cluster, req.Auth)
}

// TestConnectionInput tests the connection using ad-hoc form values (host/port/…) rather
// than a saved cluster, so the cluster editor validates exactly what's on screen.
func (a *App) TestConnectionInput(in model.ClusterInput, auth model.AuthContext) error {
	if in.Host == "" {
		return fmt.Errorf("host is required")
	}
	if in.Database == "" {
		return fmt.Errorf("database is required")
	}
	port := in.Port
	if port <= 0 {
		port = 5432
	}
	sslMode := in.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}
	cluster := model.Cluster{
		Alias:       in.Alias,
		Host:        in.Host,
		Port:        port,
		Database:    in.Database,
		Category:    in.Category,
		SSLMode:     sslMode,
		ConnectUser: in.ConnectUser,
	}
	return pg.TestConnection(cluster, auth)
}

func (a *App) RunOperation(req model.RunRequest) ([]model.ClusterResult, error) {
	return a.batch.Run(req)
}

// RunRoleBatch applies, per cluster, an ordered list of operations inside a single transaction
// (all-or-nothing per cluster). Used by the role Create/Alter forms. Emits a "role-batch-progress"
// event (model.ClusterProgress) as each cluster starts and finishes so the UI can show live status.
func (a *App) RunRoleBatch(req model.RoleBatchRequest) ([]model.ClusterResult, error) {
	return a.batch.RunRoleBatch(req, func(ev model.ClusterProgress) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "role-batch-progress", ev)
		}
	})
}

// SearchRoles scans the selected clusters/categories for roles matching the term
// (role name or comment). Per-cluster failures are returned as RoleMatch entries
// with Error set.
func (a *App) SearchRoles(req model.RoleSearchRequest) ([]model.RoleMatch, error) {
	if len(strings.TrimSpace(req.Term)) < 2 {
		return nil, fmt.Errorf("enter at least 2 characters to search")
	}
	return a.batch.SearchRoles(req.Term, req.CategoryIDs, req.ClusterIDs, req.Auth)
}

// LoadRoleDetails scans the selected clusters/categories for one login's per-cluster
// state (comment, full name, parent memberships). Per-cluster failures carry Error.
func (a *App) LoadRoleDetails(req model.RoleDetailsRequest) ([]model.ClusterRoleDetail, error) {
	if strings.TrimSpace(req.LoginName) == "" {
		return nil, fmt.Errorf("login name is required")
	}
	return a.batch.LoadRoleDetails(req.LoginName, req.CategoryIDs, req.ClusterIDs, req.Auth)
}

func (a *App) PreviewTargets(req model.RunRequest) ([]model.Cluster, error) {
	if len(req.CategoryIDs) == 0 && len(req.ClusterIDs) == 0 {
		return nil, fmt.Errorf("select at least one category or cluster")
	}
	return a.batch.ResolveClusters(req)
}
