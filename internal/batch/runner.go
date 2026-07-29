package batch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
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
	fn, args, err := commands.BuildArgs(cfg, req.OperationSpec)
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

// maxScanWorkers returns the configured concurrency (default 5).
func (r *Runner) maxScanWorkers() int {
	n := r.store.Get().Batch.MaxConcurrency
	if n <= 0 {
		return 5
	}
	return n
}

// scanClusters runs work on the given clusters concurrently.
func (r *Runner) scanClusters(clusters []model.Cluster, auth model.AuthContext, work func(ctx context.Context, cluster model.Cluster, conn *pgx.Conn) error, onError func(cluster model.Cluster, msg string)) {
	sem := make(chan struct{}, r.maxScanWorkers())
	var wg sync.WaitGroup
	for _, cluster := range clusters {
		wg.Add(1)
		go func(cl model.Cluster) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			conn, err := pg.Connect(ctx, cl, auth)
			if err != nil {
				onError(cl, err.Error())
				return
			}
			defer conn.Close(ctx)

			if err := work(ctx, cl, conn); err != nil {
				onError(cl, err.Error())
			}
		}(cluster)
	}
	wg.Wait()
}

// SearchRoles scans the selected clusters for roles matching term (name or comment).
func (r *Runner) SearchRoles(term string, categoryIDs, clusterIDs []string, auth model.AuthContext) ([]model.RoleMatch, error) {
	clusters, err := r.ResolveClusters(model.RunRequest{CategoryIDs: categoryIDs, ClusterIDs: clusterIDs})
	if err != nil {
		return nil, err
	}
	searchQuery := r.store.Get().DBReads.SearchRoles.Query

	var mu sync.Mutex
	var out []model.RoleMatch

	r.scanClusters(clusters, auth,
		func(ctx context.Context, cl model.Cluster, conn *pgx.Conn) error {
			rows, err := pg.SearchRoles(ctx, conn, searchQuery, term)
			if err != nil {
				return err
			}
			mu.Lock()
			for _, row := range rows {
				out = append(out, model.RoleMatch{
					ClusterID: cl.ID,
					Alias:     cl.Alias,
					Host:      cl.Host,
					Category:  cl.Category,
					LoginName: row.Name,
					Comment:   row.Comment,
					FullName:  pg.ParseFullName(row.Comment),
				})
			}
			mu.Unlock()
			return nil
		},
		func(cl model.Cluster, msg string) {
			mu.Lock()
			out = append(out, model.RoleMatch{
				ClusterID: cl.ID,
				Alias:     cl.Alias,
				Host:      cl.Host,
				Category:  cl.Category,
				Error:     msg,
			})
			mu.Unlock()
		},
	)
	return out, nil
}

// LoadRoleDetails scans the selected clusters for one login's per-cluster state.
func (r *Runner) LoadRoleDetails(loginName string, categoryIDs, clusterIDs []string, auth model.AuthContext) ([]model.ClusterRoleDetail, error) {
	clusters, err := r.ResolveClusters(model.RunRequest{CategoryIDs: categoryIDs, ClusterIDs: clusterIDs})
	if err != nil {
		return nil, err
	}
	reads := r.store.Get().DBReads
	detailQuery, parentsQuery := reads.RoleDetail.Query, reads.RoleParents.Query

	var mu sync.Mutex
	var out []model.ClusterRoleDetail

	r.scanClusters(clusters, auth,
		func(ctx context.Context, cl model.Cluster, conn *pgx.Conn) error {
			start := time.Now()
			exists, comment, parents, attrs, settings, err := pg.RoleDetail(ctx, conn, detailQuery, parentsQuery, loginName)
			if err != nil {
				return err
			}
			mu.Lock()
			out = append(out, model.ClusterRoleDetail{
				ClusterID:  cl.ID,
				Alias:      cl.Alias,
				Host:       cl.Host,
				Category:   cl.Category,
				Exists:     exists,
				Comment:    comment,
				FullName:   pg.ParseFullName(comment),
				Parents:    parents,
				Attributes: attrs,
				Settings:   settings,
				DurationMs: time.Since(start).Milliseconds(),
				Queries:    pg.RoleDetailQueries(detailQuery, parentsQuery, loginName),
			})
			mu.Unlock()
			return nil
		},
		func(cl model.Cluster, msg string) {
			mu.Lock()
			out = append(out, model.ClusterRoleDetail{
				ClusterID: cl.ID,
				Alias:     cl.Alias,
				Host:      cl.Host,
				Category:  cl.Category,
				Error:     msg,
			})
			mu.Unlock()
		},
	)
	return out, nil
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

	_, msg, err := pg.CallFunction(ctx, conn, fn, operation, args, r.store.Get().CommentFieldKeys()...)
	res.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.Status = "ok"
	res.Message = msg
	return res
}

// RunRoleBatch applies, per cluster, an ordered list of operations inside a single transaction
// (all-or-nothing per cluster). Clusters run concurrently (bounded by max concurrency); a failed
// cluster rolls back and is reported independently while the others proceed.
//
// The optional progress callback (nil to skip) is invoked from each cluster's goroutine with a
// "running" event when work actually starts (after acquiring a worker slot) and a "done" event when
// the result is ready. It fires concurrently from multiple goroutines, so it must be thread-safe.
func (r *Runner) RunRoleBatch(req model.RoleBatchRequest, progress func(model.ClusterProgress)) ([]model.ClusterResult, error) {
	cfg := r.store.Get()
	if err := commands.ValidateRoleBatch(cfg, req); err != nil {
		return nil, err
	}

	type target struct {
		cluster model.Cluster
		ops     []model.OperationSpec
	}
	targets := make([]target, 0, len(req.Clusters))
	resolved := make([]model.Cluster, 0, len(req.Clusters))
	for _, co := range req.Clusters {
		c, ok := r.store.ClusterByID(co.ClusterID)
		if !ok {
			return nil, fmt.Errorf("unknown cluster: %s", co.ClusterID)
		}
		targets = append(targets, target{cluster: c, ops: co.Operations})
		resolved = append(resolved, c)
	}

	if commands.RequiresProductionConfirm(cfg.Categories, resolved) && !req.ConfirmProduction {
		return nil, fmt.Errorf("production clusters selected: confirm production execution")
	}

	results := make([]model.ClusterResult, len(targets))
	sem := make(chan struct{}, r.maxScanWorkers())
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(idx int, cl model.Cluster, ops []model.OperationSpec) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if progress != nil {
				progress(model.ClusterProgress{
					ClusterID: cl.ID, Alias: cl.Alias, Host: cl.Host,
					Category: cl.Category, Phase: "running",
				})
			}
			res := r.runClusterTx(cfg, cl, ops, req.Auth)
			results[idx] = res
			if progress != nil {
				progress(model.ClusterProgress{
					ClusterID: res.ClusterID, Alias: res.Alias, Host: res.Host,
					Category: res.Category, Phase: "done",
					Status: res.Status, Message: res.Message, DurationMs: res.DurationMs,
					Queries: res.Queries,
				})
			}
		}(i, t.cluster, t.ops)
	}
	wg.Wait()
	return results, nil
}

// runClusterTx runs all of a cluster's operations in one transaction: commit on success,
// rollback (leaving nothing applied) on the first error, naming the failing operation.
func (r *Runner) runClusterTx(cfg model.Config, cluster model.Cluster, ops []model.OperationSpec, auth model.AuthContext) model.ClusterResult {
	start := time.Now()
	res := model.ClusterResult{
		ClusterID: cluster.ID,
		Alias:     cluster.Alias,
		Host:      cluster.Host,
		Category:  cluster.Category,
		Status:    "error",
	}

	// The timeout spans the whole op list + commit, so scale it with the operation count.
	timeout := 30*time.Second + time.Duration(len(ops))*10*time.Second
	if timeout > 120*time.Second {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := pg.Connect(ctx, cluster, auth)
	if err != nil {
		res.Message = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		res.Message = "begin transaction: " + err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	res.Queries = make([]string, 0, len(ops))
	for i, op := range ops {
		fn, args, berr := commands.BuildArgs(cfg, op)
		var sqlText string
		if berr == nil {
			sqlText, _, berr = pg.ExecuteOperation(ctx, tx, fn, op.Operation, args, cfg.CommentFieldKeys()...)
		}
		if sqlText != "" {
			res.Queries = append(res.Queries, sqlText) // include the failing op's SQL too
		}
		if berr != nil {
			_ = tx.Rollback(ctx)
			res.Message = fmt.Sprintf("operation %d/%d (%s): %v", i+1, len(ops), op.Operation, berr)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
	}

	if err := tx.Commit(ctx); err != nil {
		res.Message = "commit: " + err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Status = "ok"
	res.Message = fmt.Sprintf("%d operation(s) committed", len(ops))
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}
