package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

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
		// Defaults are vanilla PostgreSQL DDL (statement mode). Deployments that need
		// privileged wrapper functions override these in config / the Settings editor.
		DBFunctions: model.DBFunctions{
			CreateRole: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "CREATE ROLE ${loginname}",
			},
			RemoveRole: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "DROP ROLE ${loginname}",
			},
			GrantParents: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "GRANT ${parent_roles} TO ${loginname}",
			},
			RevokeParents: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "REVOKE ${parent_roles} FROM ${loginname}",
			},
			ChangePassword: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "ALTER ROLE ${loginname} PASSWORD ${new_password}",
			},
			SetComment: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "COMMENT ON ROLE ${loginname} IS ${comment}",
			},
			SetAttribute: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "ALTER ROLE ${loginname} WITH ${attributes}",
			},
			SetConfig: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "ALTER ROLE ${loginname} SET ${config_name} = ${config_value}",
			},
			ResetConfig: model.DBFunction{
				Execution: model.ExecutionStatement,
				Call:      "ALTER ROLE ${loginname} RESET ${config_name}",
			},
		},
		// Introspection reads. Defaults are vanilla catalog queries; result columns are
		// scanned by name against the contract documented on model.DBReads. $1 is the search
		// pattern (search_roles) or the role name (role_detail / role_parents /
		// role_dependencies).
		DBReads: model.DBReads{
			SearchRoles:      model.DBRead{Query: defaultSearchRolesQuery},
			RoleDetail:       model.DBRead{Query: defaultRoleDetailQuery},
			RoleParents:      model.DBRead{Query: defaultRoleParentsQuery},
			RoleDependencies: model.DBRead{Query: defaultRoleDependenciesQuery},
		},
		Batch:         model.BatchSettings{MaxConcurrency: 5},
		UI:            model.UISettings{Theme: model.ThemeSystem, CommentDefaultView: model.CommentViewFields, PasswordGen: model.DefaultPasswordGen()},
		CommentFields: defaultCommentFields(),
		SearchColumns: defaultSearchColumns(),
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

// defaultSearchColumns is the built-in Find-role result layout, used when config omits
// search_columns. Mirrors what the app showed before the columns became configurable.
func defaultSearchColumns() []model.SearchColumn {
	return []model.SearchColumn{
		{Label: "Full name", Template: "${{full_name}}"},
	}
}

type Store struct {
	// mu guards cfg. Wails dispatches every bound method on its own goroutine, so config
	// reads (Get/ClusterByID/…) can race writes (Save*/Update*/Add*/Delete*) without it.
	mu   sync.RWMutex
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
	migrateDBReads(&cfg.DBReads)
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
	s.mu.Lock()
	defer s.mu.Unlock()
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
	cfg.UI.PasswordGen = cfg.UI.PasswordGen.Normalized()
	cfg.ParentRoles = sanitizeParentRoles(cfg.ParentRoles)
	cfg.CommentFields = sanitizeCommentFields(cfg.CommentFields)
	if len(cfg.CommentFields) == 0 {
		cfg.CommentFields = defaultCommentFields()
	}
	// Absent key (older config) → the built-in column; an explicit `search_columns: []`
	// stays empty, because "role name only" is a legitimate choice the user can save.
	cfg.SearchColumns = sanitizeSearchColumns(cfg.SearchColumns)
	if cfg.SearchColumns == nil {
		cfg.SearchColumns = defaultSearchColumns()
	}
	migrateDBFunctions(&cfg.DBFunctions)
	migrateDBReads(&cfg.DBReads)
	s.cfg = cfg
	return nil
}

// Save marshals and atomically persists the config. Callers that already hold the write
// lock (the Update*/Add*/Delete* methods) use the unlocked save() instead to avoid a
// re-entrant deadlock.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// save persists s.cfg. The caller must hold s.mu. It writes to a temp file in the same
// directory and renames it over the target, so a crash / full disk / interrupted write
// never leaves a truncated or empty config.yaml (os.WriteFile truncates in place).
func (s *Store) save() error {
	data, err := yaml.Marshal(s.cfg)
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, data, 0o600)
}

// atomicWriteFile writes data to a sibling temp file, fsyncs it, and renames it over path.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below succeeds
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Get returns a deep-ish copy of the config: the slice fields are cloned so a caller
// iterating the result is never affected by a concurrent in-place write to s.cfg.
func (s *Store) Get() model.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.cfg)
}

func cloneConfig(c model.Config) model.Config {
	c.Categories = append([]model.Category(nil), c.Categories...)
	c.Clusters = append([]model.Cluster(nil), c.Clusters...)
	c.ParentRoles = append([]string(nil), c.ParentRoles...)
	c.CommentFields = append([]model.CommentField(nil), c.CommentFields...)
	c.SearchColumns = append([]model.SearchColumn(nil), c.SearchColumns...)
	c.Targets.CategoryIDs = append([]string(nil), c.Targets.CategoryIDs...)
	c.Targets.ClusterIDs = append([]string(nil), c.Targets.ClusterIDs...)
	return c
}

func (s *Store) ConfigPath() string {
	return s.path
}

func (s *Store) UpdateDBFunctions(fn model.DBFunctions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateDBFunctions(fn, s.cfg.CommentFieldKeys()...); err != nil {
		return err
	}
	s.cfg.DBFunctions = fn
	return s.save()
}

func (s *Store) UpdateDBReads(reads model.DBReads) error {
	migrateDBReads(&reads) // a blank query from the editor falls back to its default
	if err := validateDBReads(reads); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.DBReads = reads
	return s.save()
}

// SaveSettings validates and applies the whole Settings page in ONE write. Every field is
// validated BEFORE anything is assigned, so a rejected template can no longer leave the parent
// groups and comment fields already persisted (which is what the previous seven-call sequence
// did). Command templates are checked against the comment fields FROM THIS PAYLOAD, so adding a
// field and using it as a ${{key}} placeholder in the same save works without ordering tricks.
func (s *Store) SaveSettings(p model.SettingsPayload) error {
	parents, err := validateParentRoles(p.ParentRoles)
	if err != nil {
		return err
	}
	fields, err := validateCommentFields(p.CommentFields)
	if err != nil {
		return err
	}
	cols, err := validateSearchColumns(p.SearchColumns)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Key != "" {
			keys = append(keys, f.Key)
		}
	}
	if err := validateDBFunctions(p.DBFunctions, keys...); err != nil {
		return err
	}
	reads := p.DBReads
	migrateDBReads(&reads) // a blank query from the editor falls back to its default
	if err := validateDBReads(reads); err != nil {
		return err
	}
	batch := p.Batch
	if batch.MaxConcurrency <= 0 {
		batch.MaxConcurrency = 5
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.ParentRoles = parents
	s.cfg.CommentFields = fields
	s.cfg.SearchColumns = cols
	s.cfg.DBFunctions = p.DBFunctions
	s.cfg.DBReads = reads
	s.cfg.Batch = batch
	s.cfg.UI = normalizeUI(p.UI)
	return s.save()
}

func (s *Store) UpdateBatch(batch model.BatchSettings) error {
	if batch.MaxConcurrency <= 0 {
		batch.MaxConcurrency = 5
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Batch = batch
	return s.save()
}

// normalizeUI coerces the UI block to valid values. Field-by-field (not a blanket copy) so a
// field added to UISettings has to be considered here rather than silently riding through.
func normalizeUI(ui model.UISettings) model.UISettings {
	return model.UISettings{
		Theme:                  model.NormalizeTheme(ui.Theme),
		CommentDefaultView:     model.NormalizeCommentView(ui.CommentDefaultView),
		StageCreateOnTargetAdd: ui.StageCreateOnTargetAdd,
		CheckForUpdates:        ui.CheckForUpdates,
		PasswordGen:            ui.PasswordGen.Normalized(),
	}
}

func (s *Store) UpdateUI(ui model.UISettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.UI = normalizeUI(ui)
	return s.save()
}

// UpdateTargets persists the Operations-page target selection (cluster groups / clusters).
func (s *Store) UpdateTargets(t model.TargetSelection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Targets = t
	return s.save()
}

// SetUpdateSeenVersion records the latest release version the update popup last showed, so it
// isn't shown again for a version the user already dismissed.
func (s *Store) SetUpdateSeenVersion(v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.UpdateSeenVersion = v
	return s.save()
}

// UpdateWindowSize persists the last OS window size (restored on next launch). Ignores
// non-positive values so a transient 0 during teardown can't wipe a good size.
func (s *Store) UpdateWindowSize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if width > 0 {
		s.cfg.WindowWidth = width
	}
	if height > 0 {
		s.cfg.WindowHeight = height
	}
	return s.save()
}

func (s *Store) UpdateParentRoles(roles []string) error {
	cleaned, err := validateParentRoles(roles)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.ParentRoles = cleaned
	return s.save()
}

func (s *Store) UpdateSearchColumns(cols []model.SearchColumn) error {
	cleaned, err := validateSearchColumns(cols)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.SearchColumns = cleaned
	return s.save()
}

func (s *Store) UpdateCommentFields(fields []model.CommentField) error {
	cleaned, err := validateCommentFields(fields)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.CommentFields = cleaned
	return s.save()
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
		Password:    in.Password,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Clusters = append(s.cfg.Clusters, c)
	if err := s.save(); err != nil {
		return model.Cluster{}, err
	}
	return c, nil
}

func (s *Store) UpdateCluster(id string, in model.ClusterInput) (model.Cluster, error) {
	if err := validateClusterInput(in); err != nil {
		return model.Cluster{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
			Password:    in.Password,
		}
		if err := s.save(); err != nil {
			return model.Cluster{}, err
		}
		return s.cfg.Clusters[i], nil
	}
	return model.Cluster{}, fmt.Errorf("cluster not found: %s", id)
}

func (s *Store) DeleteCluster(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cfg.Clusters {
		if c.ID == id {
			s.cfg.Clusters = append(s.cfg.Clusters[:i], s.cfg.Clusters[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("cluster not found: %s", id)
}

func (s *Store) ClusterByID(id string) (model.Cluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Cluster
	for _, c := range s.cfg.Clusters {
		if _, ok := set[c.Category]; ok {
			out = append(out, c)
		}
	}
	return out
}

func (s *Store) CategoryByID(id string) (model.Category, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.cfg.Categories {
		if c.ID == id {
			return model.Category{}, fmt.Errorf("a group with id %q already exists", id)
		}
	}
	c := model.Category{ID: id, Label: label, Color: normalizeColor(in.Color), Confirm: in.Confirm}
	s.cfg.Categories = append(s.cfg.Categories, c)
	if err := s.save(); err != nil {
		return model.Category{}, err
	}
	return c, nil
}

func (s *Store) UpdateCategory(id string, in model.CategoryInput) (model.Category, error) {
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return model.Category{}, errors.New("label is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cfg.Categories {
		if c.ID != id {
			continue
		}
		s.cfg.Categories[i] = model.Category{ID: id, Label: label, Color: normalizeColor(in.Color), Confirm: in.Confirm}
		if err := s.save(); err != nil {
			return model.Category{}, err
		}
		return s.cfg.Categories[i], nil
	}
	return model.Category{}, fmt.Errorf("group not found: %s", id)
}

func (s *Store) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cl := range s.cfg.Clusters {
		if cl.Category == id {
			return fmt.Errorf("group %q is in use by cluster %q", id, cl.Alias)
		}
	}
	for i, c := range s.cfg.Categories {
		if c.ID == id {
			s.cfg.Categories = append(s.cfg.Categories[:i], s.cfg.Categories[i+1:]...)
			return s.save()
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
			Password: c.Password,
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
			ConnectUser: c.ConnectUser, Password: c.Password,
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Categories = cats
	s.cfg.Clusters = out
	return s.save()
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

// searchBuiltins is the closed bare-${…} namespace for a search-column template. A comment key is
// written ${{key}} instead, so a key named "comment" stays reachable.
var searchBuiltins = map[string]bool{"comment": true}

// searchTokenRE mirrors calltemplate's token regex: ${{key}} is a comment key, ${name} a built-in.
// Both name classes exclude braces, so the bare branch cannot swallow a ${{…}} token.
var searchTokenRE = regexp.MustCompile(`\$\{\{([^{}]*)\}\}|\$\{([^{}]*)\}`)

// validateSearchColumns trims label+template and drops rows with a blank template (a column
// with nothing to show). A blank label is kept — the header cell is then simply empty. A malformed
// or unknown-built-in placeholder is rejected, so a typo is reported rather than rendering empty.
func validateSearchColumns(cols []model.SearchColumn) ([]model.SearchColumn, error) {
	var out []model.SearchColumn
	for _, c := range cols {
		tmpl := strings.TrimSpace(c.Template)
		if tmpl == "" {
			continue
		}
		if err := checkSearchTemplate(tmpl); err != nil {
			return nil, err
		}
		out = append(out, model.SearchColumn{Label: strings.TrimSpace(c.Label), Template: tmpl})
	}
	return out, nil
}

// sanitizeSearchColumns is the load-time counterpart: it drops a structurally broken row rather
// than failing, but keeps a row whose only problem is an unknown bare name — that one still renders
// (as the literal token) and the user gets the guiding error when they next save, which is better
// than silently deleting a column from their config file. Nil-ness is preserved, so Load can tell
// an absent key (→ default column) from an explicit empty list.
func sanitizeSearchColumns(cols []model.SearchColumn) []model.SearchColumn {
	if cols == nil {
		return nil
	}
	out := make([]model.SearchColumn, 0, len(cols))
	for _, c := range cols {
		tmpl := strings.TrimSpace(c.Template)
		if tmpl == "" || checkSearchTemplateSyntax(tmpl) != nil {
			continue
		}
		out = append(out, model.SearchColumn{Label: strings.TrimSpace(c.Label), Template: tmpl})
	}
	return out
}

// checkSearchTemplateSyntax rejects a malformed placeholder — a "${" that is not part of a
// well-formed ${name} / ${{name}} token, or an empty name. Comment KEYS are deliberately
// unconstrained beyond that: they are arbitrary JSON keys (which may contain '-' or '.'), and the
// template is display text the frontend HTML-escapes, never SQL.
func checkSearchTemplateSyntax(tmpl string) error {
	if strings.Contains(searchTokenRE.ReplaceAllString(tmpl, ""), "${") {
		return fmt.Errorf(
			"invalid search column template %q: unfinished placeholder — write ${comment} or ${{comment_key}} with both braces closed", tmpl)
	}
	for _, m := range searchTokenRE.FindAllStringSubmatch(tmpl, -1) {
		if strings.TrimSpace(m[1]+m[2]) == "" {
			return fmt.Errorf("invalid search column template %q: empty placeholder — put a name between the braces", tmpl)
		}
	}
	return nil
}

// checkSearchTemplate is checkSearchTemplateSyntax plus the closed bare-namespace rule, used when
// saving. The error names the fix, because writing ${full_name} for a comment key is the mistake
// this separation invites. It deliberately does NOT call ${comment} a "built-in": the call-template
// built-ins (${loginname}, ${parent_roles}, …) are not available here at all, so that framing sent
// users looking for placeholders a search column cannot have.
func checkSearchTemplate(tmpl string) error {
	if err := checkSearchTemplateSyntax(tmpl); err != nil {
		return err
	}
	for _, m := range searchTokenRE.FindAllStringSubmatch(tmpl, -1) {
		if m[1] != "" { // a ${{key}} token — any comment key is allowed
			continue
		}
		name := strings.TrimSpace(m[2])
		if !searchBuiltins[name] {
			return fmt.Errorf(
				"invalid search column template %q: ${%s} is not supported — use ${{%s}} for a comment key, or ${comment} for the whole comment",
				tmpl, name, name)
		}
	}
	return nil
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
