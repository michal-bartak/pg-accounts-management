package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/batch"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/model"
	"github.com/michalbartak/dbaccounts/internal/pg"
	"github.com/michalbartak/dbaccounts/internal/update"
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

// GetClustersPath returns the clusters.yaml path (clusters, groups, target selection), shown in
// Settings beside the config path.
func (a *App) GetClustersPath() string {
	return a.store.ClustersPath()
}

// GetDefaultTemplates returns the built-in call templates and introspection queries — the same
// values a fresh config is seeded with. The Settings editor's "Default" button reverts a template
// to these, so the frontend does NOT keep its own copy of the SQL: the defaults live in
// config.DefaultConfig() alone and a change there reaches the button with no second edit.
func (a *App) GetDefaultTemplates() model.DefaultTemplates {
	def := config.DefaultConfig()
	return model.DefaultTemplates{DBFunctions: def.DBFunctions, DBReads: def.DBReads}
}

// GetAppVersion returns build metadata (version, git commit, build date).
func (a *App) GetAppVersion() model.AppVersion {
	i := version.Get()
	return model.AppVersion{
		Version:   i.Version,
		Commit:    i.Commit,
		BuildDate: i.BuildDate,
		RepoURL:   i.RepoURL,
		DocsURL:   i.DocsURL,
	}
}

// CheckForUpdate looks up the project's latest published GitHub Release and compares it against
// the running version. Best-effort: a repo with no releases reports "up to date"; network errors
// surface to the caller (the UI reports them without blocking).
func (a *App) CheckForUpdate() (model.UpdateInfo, error) {
	v := version.Get()
	return update.Check(a.ctx, v.Version, v.RepoURL)
}

// SetUpdateSeenVersion records the release version the startup popup last showed, so it isn't
// shown again for a version the user already dismissed.
func (a *App) SetUpdateSeenVersion(v string) error {
	return a.store.SetUpdateSeenVersion(v)
}

// GetPendingUpdate returns — without any network call — the update the user was last informed
// about (the persisted UpdateSeenVersion) if it is still newer than the running version. This
// drives the "update available" badge across restarts, including when the startup auto-check is
// off; it naturally reports "not available" once the user upgrades past the seen version.
func (a *App) GetPendingUpdate() (model.UpdateInfo, error) {
	v := version.Get()
	return update.Pending(a.store.Get().UpdateSeenVersion, v.Version, v.RepoURL), nil
}

func (a *App) ReloadConfig() (model.Config, error) {
	if err := a.store.Load(); err != nil {
		return model.Config{}, err
	}
	return a.store.Get(), nil
}

// SaveSettings persists the whole Settings page atomically: everything validates before anything
// is written, so a rejected value leaves the saved config exactly as it was. Replaces the previous
// sequence of SaveParentRoles → SaveCommentFields → SaveDBFunctions → … calls, each of which wrote
// the config file separately and could leave it half-updated.
func (a *App) SaveSettings(p model.SettingsPayload) error {
	return a.store.SaveSettings(p)
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

// SaveSearchColumns persists the extra columns shown next to the role name in the Find-role
// results. An empty list is valid and means "role name only".
func (a *App) SaveSearchColumns(cols []model.SearchColumn) error {
	return a.store.UpdateSearchColumns(cols)
}

// SaveTargetSelection persists the Operations-page target selection so it survives
// re-renders (e.g. after saving Settings) and app restarts.
func (a *App) SaveTargetSelection(t model.TargetSelection) error {
	return a.store.UpdateTargets(t)
}

// SaveWindowSize persists the current OS window size so it can be restored on next launch.
// The frontend passes Wails' WindowGetSize (the OS window size, NOT the webview viewport) —
// otherwise the window would shrink by the chrome height on every launch.
func (a *App) SaveWindowSize(width, height int) error {
	return a.store.UpdateWindowSize(width, height)
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
		Password:    in.Password,
	}
	return pg.TestConnection(cluster, auth)
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
// (role name or comment). One entry per cluster; a per-cluster failure carries Error.
func (a *App) SearchRoles(req model.RoleSearchRequest) ([]model.ClusterRoleMatches, error) {
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

// LoadRoleDependencies runs the pre-flight dependency check (objects that depend on the role)
// on the selected clusters, before the role is dropped there. Per-cluster failures carry Error.
func (a *App) LoadRoleDependencies(req model.RoleDependenciesRequest) ([]model.ClusterRoleDependencies, error) {
	if strings.TrimSpace(req.LoginName) == "" {
		return nil, fmt.Errorf("login name is required")
	}
	return a.batch.LoadRoleDependencies(req.LoginName, req.CategoryIDs, req.ClusterIDs, req.Auth)
}

func (a *App) PreviewTargets(req model.RunRequest) ([]model.Cluster, error) {
	if len(req.CategoryIDs) == 0 && len(req.ClusterIDs) == 0 {
		return nil, fmt.Errorf("select at least one category or cluster")
	}
	return a.batch.ResolveClusters(req)
}
