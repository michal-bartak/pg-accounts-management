package batch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/michalbartak/dbaccounts/internal/commands"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/model"
	"github.com/michalbartak/dbaccounts/internal/pg"
)

type Runner struct {
	store *config.Store
}

func NewRunner(store *config.Store) *Runner {
	return &Runner{store: store}
}

func (r *Runner) ResolveClusters(req model.RunRequest) ([]model.Cluster, error) {
	seen := make(map[string]struct{})
	var out []model.Cluster

	add := func(c model.Cluster) {
		if _, ok := seen[c.ID]; ok {
			return
		}
		seen[c.ID] = struct{}{}
		out = append(out, c)
	}

	for _, id := range req.ClusterIDs {
		if c, ok := r.store.ClusterByID(id); ok {
			add(c)
		}
	}
	for _, c := range r.store.ClustersByCategories(req.CategoryIDs) {
		add(c)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no clusters matched the selection")
	}
	return out, nil
}

func (r *Runner) Run(req model.RunRequest) ([]model.ClusterResult, error) {
	cfg := r.store.Get()
	if err := commands.ValidateRequest(cfg, req); err != nil {
		return nil, err
	}

	clusters, err := r.ResolveClusters(req)
	if err != nil {
		return nil, err
	}

	if commands.RequiresProductionConfirm(cfg.Categories, clusters) && !req.ConfirmProduction {
		return nil, fmt.Errorf("production clusters selected: confirm production execution")
	}
	fn, args, err := commands.BuildArgs(cfg, req)
	if err != nil {
		return nil, err
	}

	maxWorkers := cfg.Batch.MaxConcurrency
	if maxWorkers <= 0 {
		maxWorkers = 5
	}

	results := make([]model.ClusterResult, len(clusters))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, cluster := range clusters {
		wg.Add(1)
		go func(idx int, cl model.Cluster) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = r.runOne(cl, req.Operation, fn, args, req.Auth)
		}(i, cluster)
	}
	wg.Wait()
	return results, nil
}

func (r *Runner) runOne(cluster model.Cluster, operation string, fn model.DBFunction, args map[string]string, auth model.AuthContext) model.ClusterResult {
	start := time.Now()
	res := model.ClusterResult{
		ClusterID: cluster.ID,
		Alias:     cluster.Alias,
		Host:      cluster.Host,
		Category:  cluster.Category,
		Status:    "error",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pg.Connect(ctx, cluster, auth)
	if err != nil {
		res.Message = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer conn.Close(ctx)

	msg, err := pg.CallFunction(ctx, conn, fn, operation, args)
	res.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.Status = "ok"
	res.Message = msg
	return res
}
