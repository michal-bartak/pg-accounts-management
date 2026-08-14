package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// configFileName holds the app configuration: templates, comment fields, UI preferences.
	configFileName = "config.yaml"
	// clustersFileName holds the cluster inventory, its groups, and the target selection.
	// It sits next to config.yaml; Store derives its path from the main one.
	clustersFileName = "clusters.yaml"
)

func ConfigDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support", "DbAccounts")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = home
		}
		base = filepath.Join(appData, "DbAccounts")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config", "dbaccounts")
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return base, nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}
