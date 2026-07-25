package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/michalbartak/dbaccounts/internal/model"
	"gopkg.in/yaml.v3"
)

// parentRoleRE bounds a preconfigured parent group to a bare SQL identifier, matching
// what the grant path accepts (unquoted role names).
var parentRoleRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var ErrNotFound = errors.New("config not found")

func DefaultConfig() model.Config {
	return model.Config{
		Version: 1,
		Categories: []model.Category{
			{ID: "production", Label: "Production", Color: "#e8a838", Confirm: true},
			{ID: "uat", Label: "UAT", Color: "#6eb5ff", Confirm: false},
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
			SetComment: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "COMMENT ON ROLE ${loginname} IS ${comment}",
			},
			SetAttribute: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "ALTER ROLE ${loginname} WITH ${attribute}",
			},
		},
		Batch:         model.BatchSettings{MaxConcurrency: 5},
		UI:            model.UISettings{Theme: model.ThemeSystem, CommentDefaultView: model.CommentViewFields},
		CommentFields: defaultCommentFields(),
	}
}

// defaultCommentFields is the built-in comment-key mapping (the company convention),
// used when config omits comment_fields.
func defaultCommentFields() []model.CommentField {
	return []model.CommentField{
		{Key: "full_name", Label: "Full name"},
		{Key: "e_mail", Label: "Email"},
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
	cfg.UI.Theme = model.NormalizeTheme(cfg.UI.Theme)
	cfg.UI.CommentDefaultView = model.NormalizeCommentView(cfg.UI.CommentDefaultView)
	cfg.ParentRoles = sanitizeParentRoles(cfg.ParentRoles)
	cfg.CommentFields = sanitizeCommentFields(cfg.CommentFields)
	if len(cfg.CommentFields) == 0 {
		cfg.CommentFields = defaultCommentFields()
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

func (s *Store) UpdateUI(ui model.UISettings) error {
	s.cfg.UI = model.UISettings{
		Theme:              model.NormalizeTheme(ui.Theme),
		CommentDefaultView: model.NormalizeCommentView(ui.CommentDefaultView),
	}
	return s.Save()
}

// UpdateTargets persists the Operations-page target selection (cluster groups / clusters).
func (s *Store) UpdateTargets(t model.TargetSelection) error {
	s.cfg.Targets = t
	return s.Save()
}

func (s *Store) UpdateParentRoles(roles []string) error {
	cleaned, err := validateParentRoles(roles)
	if err != nil {
		return err
	}
	s.cfg.ParentRoles = cleaned
	return s.Save()
}

func (s *Store) UpdateCommentFields(fields []model.CommentField) error {
	cleaned, err := validateCommentFields(fields)
	if err != nil {
		return err
	}
	s.cfg.CommentFields = cleaned
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

func (s *Store) CategoryByID(id string) (model.Category, bool) {
	for _, c := range s.cfg.Categories {
		if c.ID == id {
			return c, true
		}
	}
	return model.Category{}, false
}

func (s *Store) AddCategory(in model.CategoryInput) (model.Category, error) {
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return model.Category{}, errors.New("label is required")
	}
	id := slugify(label)
	if id == "" {
		return model.Category{}, errors.New("label must contain a letter or digit")
	}
	for _, c := range s.cfg.Categories {
		if c.ID == id {
			return model.Category{}, fmt.Errorf("a group with id %q already exists", id)
		}
	}
	c := model.Category{ID: id, Label: label, Color: normalizeColor(in.Color), Confirm: in.Confirm}
	s.cfg.Categories = append(s.cfg.Categories, c)
	if err := s.Save(); err != nil {
		return model.Category{}, err
	}
	return c, nil
}

func (s *Store) UpdateCategory(id string, in model.CategoryInput) (model.Category, error) {
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return model.Category{}, errors.New("label is required")
	}
	for i, c := range s.cfg.Categories {
		if c.ID != id {
			continue
		}
		s.cfg.Categories[i] = model.Category{ID: id, Label: label, Color: normalizeColor(in.Color), Confirm: in.Confirm}
		if err := s.Save(); err != nil {
			return model.Category{}, err
		}
		return s.cfg.Categories[i], nil
	}
	return model.Category{}, fmt.Errorf("group not found: %s", id)
}

func (s *Store) DeleteCategory(id string) error {
	for _, cl := range s.cfg.Clusters {
		if cl.Category == id {
			return fmt.Errorf("group %q is in use by cluster %q", id, cl.Alias)
		}
	}
	for i, c := range s.cfg.Categories {
		if c.ID == id {
			s.cfg.Categories = append(s.cfg.Categories[:i], s.cfg.Categories[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("group not found: %s", id)
}

// SaveClustersAndCategories replaces the whole clusters+categories set in one atomic write
// (used by the staged Clusters editor). Categories: label required, id = slugify(label) when
// empty else kept, duplicate ids rejected, colour normalized. Clusters: validated, port/sslmode
// defaulted, a fresh UUID minted when id is empty else kept. Referential integrity: every
// cluster's category must exist in the resulting category set. Nothing is persisted unless the
// whole set validates.
func (s *Store) SaveClustersAndCategories(clusters []model.Cluster, categories []model.Category) error {
	cats := make([]model.Category, 0, len(categories))
	seenCat := make(map[string]bool, len(categories))
	for _, c := range categories {
		label := strings.TrimSpace(c.Label)
		if label == "" {
			return errors.New("group label is required")
		}
		id := strings.TrimSpace(c.ID)
		if id == "" {
			id = slugify(label)
		}
		if id == "" {
			return fmt.Errorf("group %q label must contain a letter or digit", label)
		}
		if seenCat[id] {
			return fmt.Errorf("a group with id %q already exists", id)
		}
		seenCat[id] = true
		cats = append(cats, model.Category{ID: id, Label: label, Color: normalizeColor(c.Color), Confirm: c.Confirm})
	}
	if len(cats) == 0 {
		return errors.New("at least one cluster group is required")
	}

	out := make([]model.Cluster, 0, len(clusters))
	for _, c := range clusters {
		in := model.ClusterInput{
			Alias: c.Alias, Host: c.Host, Port: c.Port, Database: c.Database,
			Category: c.Category, SSLMode: c.SSLMode, ConnectUser: c.ConnectUser,
		}
		if err := validateClusterInput(in); err != nil {
			return err
		}
		if !seenCat[c.Category] {
			return fmt.Errorf("cluster %q references unknown group %q", c.Alias, c.Category)
		}
		id := strings.TrimSpace(c.ID)
		if id == "" {
			id = uuid.New().String()
		}
		out = append(out, model.Cluster{
			ID: id, Alias: c.Alias, Host: c.Host, Port: defaultPort(c.Port),
			Database: c.Database, Category: c.Category, SSLMode: defaultSSLMode(c.SSLMode),
			ConnectUser: c.ConnectUser,
		})
	}

	s.cfg.Categories = cats
	s.cfg.Clusters = out
	return s.Save()
}

// slugify turns a label into a lowercase [a-z0-9_] id.
func slugify(label string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if b.Len() > 0 && !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

// validateParentRoles trims, drops empties, dedupes (order-preserving) and rejects any
// entry that is not a bare identifier.
func validateParentRoles(roles []string) ([]string, error) {
	seen := make(map[string]bool, len(roles))
	var out []string
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !parentRoleRE.MatchString(r) {
			return nil, fmt.Errorf("invalid parent group %q: use letters, digits, underscore", r)
		}
		if seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out, nil
}

// sanitizeParentRoles trims/drops empties/dedupes without failing (used on load, to
// tolerate hand-edited config).
func sanitizeParentRoles(roles []string) []string {
	seen := make(map[string]bool, len(roles))
	var out []string
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// validateCommentFields trims key+label, drops empty-key rows, dedupes by key
// (order-preserving), rejects keys that are not bare identifiers, and defaults a blank
// label to the key.
func validateCommentFields(fields []model.CommentField) ([]model.CommentField, error) {
	seen := make(map[string]bool, len(fields))
	var out []model.CommentField
	for _, f := range fields {
		key := strings.TrimSpace(f.Key)
		if key == "" {
			continue
		}
		if !parentRoleRE.MatchString(key) {
			return nil, fmt.Errorf("invalid comment field key %q: use letters, digits, underscore", key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		label := strings.TrimSpace(f.Label)
		if label == "" {
			label = key
		}
		out = append(out, model.CommentField{Key: key, Label: label})
	}
	return out, nil
}

// sanitizeCommentFields trims/drops empty-key rows/dedupes without failing (used on load,
// to tolerate hand-edited config). Rows with an invalid key are dropped rather than rejected.
func sanitizeCommentFields(fields []model.CommentField) []model.CommentField {
	seen := make(map[string]bool, len(fields))
	var out []model.CommentField
	for _, f := range fields {
		key := strings.TrimSpace(f.Key)
		if key == "" || seen[key] || !parentRoleRE.MatchString(key) {
			continue
		}
		seen[key] = true
		label := strings.TrimSpace(f.Label)
		if label == "" {
			label = key
		}
		out = append(out, model.CommentField{Key: key, Label: label})
	}
	return out
}

// normalizeColor keeps a #rrggbb hex or returns "" (frontend falls back to a default).
func normalizeColor(c string) string {
	c = strings.TrimSpace(c)
	if len(c) == 7 && c[0] == '#' {
		return strings.ToLower(c)
	}
	return ""
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
