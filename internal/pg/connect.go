package pg

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/michal-bartak/pgcowboy/internal/model"
)

const defaultTimeout = 30 * time.Second

// pgxAttemptStart matches the start of one pgx dial-attempt segment inside a connect error —
// " host:port (resolved-addr) (" — which is how pgx separates the attempts it made.
var pgxAttemptStart = regexp.MustCompile(`\s\S+:\d+ \(`)

// splitAttempts breaks a connect error's tail into pgx's per-attempt segments, or returns nil
// when the text does not look like a multi-attempt pgx error. Input must already be
// whitespace-normalized, so each separator is exactly one space.
func splitAttempts(rest string) []string {
	locs := pgxAttemptStart.FindAllStringIndex(rest, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]string, 0, len(locs)+1)
	prev := 0
	for _, loc := range locs {
		out = append(out, rest[prev:loc[0]])
		prev = loc[0] + 1 // skip the single separating space
	}
	return append(out, rest[prev:])
}

// cleanConnectError tidies a pgx connect error for display. With sslmode=prefer (our default) pgx
// dials every resolved address twice — once with TLS, once plaintext — so a TCP-level failure
// (e.g. connection refused) reports each per-address error twice. The repeats are NOT adjacent: a
// host resolving to both IPv6 and IPv4 yields "A B A B", which is what a dead localhost port
// actually produces. Whitespace is normalized (pgx separates attempts with newlines/tabs) and
// duplicate attempts are dropped keeping first-seen order, so "A B A B" reads "A B" — one line per
// address, genuinely different failures all kept.
func cleanConnectError(msg string) string {
	msg = strings.Join(strings.Fields(msg), " ") // collapse newlines/tabs/runs of spaces
	const marker = "`: "                         // end of the `host=… user=… database=…` prefix
	i := strings.Index(msg, marker)
	if i < 0 {
		return msg
	}
	head, rest := msg[:i+len(marker)], msg[i+len(marker):]
	if segs := splitAttempts(rest); len(segs) > 1 {
		seen := make(map[string]bool, len(segs))
		uniq := make([]string, 0, len(segs))
		for _, s := range segs {
			if !seen[s] {
				seen[s] = true
				uniq = append(uniq, s)
			}
		}
		return head + strings.Join(uniq, " ")
	}
	// Unrecognised shape: fall back to collapsing an exactly-repeated tail ("A A").
	if n := len(rest); n > 1 && n%2 == 1 && rest[n/2] == ' ' && rest[:n/2] == rest[n/2+1:] {
		return head + rest[:n/2]
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
