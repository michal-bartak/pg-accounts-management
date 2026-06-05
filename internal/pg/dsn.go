package pg

import (
	"fmt"
	"net/url"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func BuildDSN(cluster model.Cluster, user, password string) string {
	var userInfo *url.Userinfo
	if password == "" {
		userInfo = url.User(user)
	} else {
		userInfo = url.UserPassword(user, password)
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   fmt.Sprintf("%s:%d", cluster.Host, cluster.Port),
		Path:   cluster.Database,
	}
	q := u.Query()
	sslmode := cluster.SSLMode
	if sslmode == "" {
		sslmode = "prefer"
	}
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
}
