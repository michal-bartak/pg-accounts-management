package pg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/michalbartak/dbaccounts/internal/model"
)

const defaultTimeout = 30 * time.Second

// cleanConnectError tidies a pgx connect error for display. With sslmode=prefer (our default)
// pgx dials the host twice — once with TLS, once plaintext — so a TCP-level failure (e.g.
// connection refused) reports the identical per-address error twice. Whitespace is normalized
// (pgx separates attempts with newlines/tabs) and an exactly-repeated tail is collapsed to one.
func cleanConnectError(msg string) string {
	msg = strings.Join(strings.Fields(msg), " ") // collapse newlines/tabs/runs of spaces
	const marker = "`: "                          // end of the `host=… user=… database=…` prefix
	i := strings.Index(msg, marker)
	if i < 0 {
		return msg
	}
	head, rest := msg[:i+len(marker)], msg[i+len(marker):]
	if n := len(rest); n > 1 && n%2 == 1 && rest[n/2] == ' ' && rest[:n/2] == rest[n/2+1:] {
		return head + rest[:n/2] // rest is "A A" (two identical attempts) → keep one
	}
	return msg
}

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
		return nil, fmt.Errorf("connect to %s: %s", cluster.Alias, cleanConnectError(err.Error()))
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
