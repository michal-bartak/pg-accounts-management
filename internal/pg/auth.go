package pg

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func ResolveUser(cluster model.Cluster, auth model.AuthContext) (string, error) {
	if cluster.ConnectUser != "" {
		return cluster.ConnectUser, nil
	}
	if auth.User != "" {
		return auth.User, nil
	}
	if u := os.Getenv("PGUSER"); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("database user not set: set connect user on cluster, PGUSER, or provide user in the run dialog")
}

func ResolvePassword(cluster model.Cluster, user string, auth model.AuthContext) (string, error) {
	if auth.Password != "" {
		return auth.Password, nil
	}
	if _, ok := os.LookupEnv("PGPASSWORD"); ok {
		return os.Getenv("PGPASSWORD"), nil
	}
	if p, found, err := lookupPgPass(cluster.Host, cluster.Port, cluster.Database, user); err != nil {
		return "", err
	} else if found {
		return p, nil
	}
	// No password — same as psql without -W (trust auth, empty password, etc.).
	return "", nil
}

func lookupPgPass(host string, port int, database, user string) (password string, found bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	path := fmt.Sprintf("%s/.pgpass", home)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	defer f.Close()

	portStr := strconv.Itoa(port)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 5 {
			continue
		}
		if !matchField(parts[0], host) || !matchField(parts[1], portStr) || !matchField(parts[2], database) || !matchField(parts[3], user) {
			continue
		}
		return parts[4], true, nil
	}
	return "", false, scanner.Err()
}

func matchField(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == value
}
