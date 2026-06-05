package main

import (
	"context"
	"fmt"

	"github.com/michalbartak/dbaccounts/internal/batch"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/envimport"
	"github.com/michalbartak/dbaccounts/internal/model"
	"github.com/michalbartak/dbaccounts/internal/pg"
	"github.com/michalbartak/dbaccounts/internal/version"
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

func (a *App) SaveBatchSettings(batch model.BatchSettings) error {
	return a.store.UpdateBatch(batch)
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

func (a *App) RunOperation(req model.RunRequest) ([]model.ClusterResult, error) {
	return a.batch.Run(req)
}

func (a *App) PreviewTargets(req model.RunRequest) ([]model.Cluster, error) {
	if len(req.CategoryIDs) == 0 && len(req.ClusterIDs) == 0 {
		return nil, fmt.Errorf("select at least one category or cluster")
	}
	return a.batch.ResolveClusters(req)
}
