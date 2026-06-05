package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/michalbartak/dbaccounts/internal/model"
	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("config not found")

func DefaultConfig() model.Config {
	return model.Config{
		Version: 1,
		Categories: []model.Category{
			{ID: "production", Label: "Production"},
			{ID: "uat", Label: "UAT"},
		},
		Clusters: []model.Cluster{},
		DBFunctions: model.DBFunctions{
			CreateRole: model.DBFunction{
				Call: "admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})",
			},
			RemoveRole: model.DBFunction{
				Call: "your_schema.remove_app_role(${loginname})",
			},
			GrantParents: model.DBFunction{
				Call: "your_schema.grant_role_parents(${loginname}, ${parent_roles})",
			},
			RevokeParents: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "REVOKE ${parent_roles} FROM ${loginname}",
			},
			ChangePassword: model.DBFunction{
				Call: "your_schema.change_role_password(${loginname}, ${new_password})",
			},
		},
		Batch: model.BatchSettings{MaxConcurrency: 5},
	}
}

type Store struct {
	path string
	cfg  model.Config
}

// NewStoreFromConfig returns a store backed by the given config (for tests and tooling).
func NewStoreFromConfig(cfg model.Config) *Store {
	if cfg.Batch.MaxConcurrency <= 0 {
		cfg.Batch.MaxConcurrency = 5
	}
	if len(cfg.Categories) == 0 {
		cfg.Categories = DefaultConfig().Categories
	}
	migrateDBFunctions(&cfg.DBFunctions)
	return &Store{path: "", cfg: cfg}
}

func NewStore() (*Store, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	s := &Store{path: path}
	if err := s.Load(); err != nil {
		if errors.Is(err, ErrNotFound) {
			s.cfg = DefaultConfig()
			if saveErr := s.Save(); saveErr != nil {
				return nil, saveErr
			}
			return s, nil
		}
		return nil, err
	}
	return s, nil
}

func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if len(cfg.Categories) == 0 {
		cfg.Categories = DefaultConfig().Categories
	}
	if cfg.Batch.MaxConcurrency <= 0 {
		cfg.Batch.MaxConcurrency = 5
	}
	migrateDBFunctions(&cfg.DBFunctions)
	s.cfg = cfg
	return nil
}

func (s *Store) Save() error {
	data, err := yaml.Marshal(s.cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Get() model.Config {
	return s.cfg
}

func (s *Store) ConfigPath() string {
	return s.path
}

func (s *Store) UpdateDBFunctions(fn model.DBFunctions) error {
	if err := validateDBFunctions(fn); err != nil {
		return err
	}
	s.cfg.DBFunctions = fn
	return s.Save()
}

func (s *Store) UpdateBatch(batch model.BatchSettings) error {
	if batch.MaxConcurrency <= 0 {
		batch.MaxConcurrency = 5
	}
	s.cfg.Batch = batch
	return s.Save()
}

func (s *Store) AddCluster(in model.ClusterInput) (model.Cluster, error) {
	if err := validateClusterInput(in); err != nil {
		return model.Cluster{}, err
	}
	c := model.Cluster{
		ID:          uuid.New().String(),
		Alias:       in.Alias,
		Host:        in.Host,
		Port:        defaultPort(in.Port),
		Database:    in.Database,
		Category:    in.Category,
		SSLMode:     defaultSSLMode(in.SSLMode),
		ConnectUser: in.ConnectUser,
	}
	s.cfg.Clusters = append(s.cfg.Clusters, c)
	if err := s.Save(); err != nil {
		return model.Cluster{}, err
	}
	return c, nil
}

func (s *Store) UpdateCluster(id string, in model.ClusterInput) (model.Cluster, error) {
	if err := validateClusterInput(in); err != nil {
		return model.Cluster{}, err
	}
	for i, c := range s.cfg.Clusters {
		if c.ID != id {
			continue
		}
		s.cfg.Clusters[i] = model.Cluster{
			ID:          id,
			Alias:       in.Alias,
			Host:        in.Host,
			Port:        defaultPort(in.Port),
			Database:    in.Database,
			Category:    in.Category,
			SSLMode:     defaultSSLMode(in.SSLMode),
			ConnectUser: in.ConnectUser,
		}
		if err := s.Save(); err != nil {
			return model.Cluster{}, err
		}
		return s.cfg.Clusters[i], nil
	}
	return model.Cluster{}, fmt.Errorf("cluster not found: %s", id)
}

func (s *Store) DeleteCluster(id string) error {
	for i, c := range s.cfg.Clusters {
		if c.ID == id {
			s.cfg.Clusters = append(s.cfg.Clusters[:i], s.cfg.Clusters[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("cluster not found: %s", id)
}

func (s *Store) ClusterByID(id string) (model.Cluster, bool) {
	for _, c := range s.cfg.Clusters {
		if c.ID == id {
			return c, true
		}
	}
	return model.Cluster{}, false
}

func (s *Store) ClustersByCategories(categoryIDs []string) []model.Cluster {
	if len(categoryIDs) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(categoryIDs))
	for _, id := range categoryIDs {
		set[id] = struct{}{}
	}
	var out []model.Cluster
	for _, c := range s.cfg.Clusters {
		if _, ok := set[c.Category]; ok {
			out = append(out, c)
		}
	}
	return out
}

func validateClusterInput(in model.ClusterInput) error {
	if in.Alias == "" {
		return errors.New("alias is required")
	}
	if in.Host == "" {
		return errors.New("host is required")
	}
	if in.Database == "" {
		return errors.New("database is required")
	}
	if in.Category == "" {
		return errors.New("category is required")
	}
	return nil
}

func defaultPort(p int) int {
	if p <= 0 {
		return 5432
	}
	return p
}

func defaultSSLMode(mode string) string {
	if mode == "" {
		return "prefer"
	}
	return mode
}
