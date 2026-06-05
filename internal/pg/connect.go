package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/michalbartak/dbaccounts/internal/model"
)

const defaultTimeout = 30 * time.Second

func Connect(ctx context.Context, cluster model.Cluster, auth model.AuthContext) (*pgx.Conn, error) {
	user, err := ResolveUser(cluster, auth)
	if err != nil {
		return nil, err
	}
	password, err := ResolvePassword(cluster, user, auth)
	if err != nil {
		return nil, err
	}
	dsn := BuildDSN(cluster, user, password)
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", cluster.Alias, err)
	}
	return conn, nil
}

func TestConnection(cluster model.Cluster, auth model.AuthContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	conn, err := Connect(ctx, cluster, auth)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return conn.Ping(ctx)
}
