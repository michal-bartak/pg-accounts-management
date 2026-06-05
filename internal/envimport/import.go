package envimport

import (
	"os"
	"strconv"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func FromEnvironment() model.EnvImport {
	port := 5432
	if p := os.Getenv("PGPORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	return model.EnvImport{
		Host:     os.Getenv("PGHOST"),
		Port:     port,
		Database: os.Getenv("PGDATABASE"),
		User:     os.Getenv("PGUSER"),
	}
}
