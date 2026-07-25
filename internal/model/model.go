package model

import "strings"

// DB function execution modes (config db_functions.<op>.execution).
const (
	ExecutionFunction  = "function"
	ExecutionStatement = "statement"
	ExecutionBlock     = "block"
)

// NormalizeExecution returns a valid execution mode; unknown values become function.
func NormalizeExecution(execution string) string {
	switch strings.ToLower(strings.TrimSpace(execution)) {
	case ExecutionStatement:
		return ExecutionStatement
	case ExecutionBlock:
		return ExecutionBlock
	default:
		return ExecutionFunction
	}
}

type Category struct {
	ID      string `yaml:"id" json:"id"`
	Label   string `yaml:"label" json:"label"`
	Color   string `yaml:"color,omitempty" json:"color,omitempty"`     // base hex, e.g. #e8a838
	Confirm bool   `yaml:"confirm,omitempty" json:"confirm,omitempty"` // require confirm popup before running
}

type CategoryInput struct {
	Label   string `json:"label"`
	Color   string `json:"color"`
	Confirm bool   `json:"confirm"`
}

type Cluster struct {
	ID          string `yaml:"id" json:"id"`
	Alias       string `yaml:"alias" json:"alias"`
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Database    string `yaml:"database" json:"database"`
	Category    string `yaml:"category" json:"category"`
	SSLMode     string `yaml:"sslmode,omitempty" json:"sslmode,omitempty"`
	ConnectUser string `yaml:"connect_user,omitempty" json:"connectUser,omitempty"`
}

type DBFunction struct {
	Call      string   `yaml:"call" json:"call"`
	Execution string   `yaml:"execution,omitempty" json:"execution,omitempty"` // function | statement | block
	Name      string   `yaml:"name,omitempty" json:"name,omitempty"`           // deprecated, migrated on load
	Params    []string `yaml:"params,omitempty" json:"params,omitempty"`       // deprecated
}

type DBFunctions struct {
	CreateRole     DBFunction `yaml:"create_role" json:"createRole"`
	RemoveRole     DBFunction `yaml:"remove_role" json:"removeRole"`
	GrantParents   DBFunction `yaml:"grant_parents" json:"grantParents"`
	RevokeParents  DBFunction `yaml:"revoke_parents" json:"revokeParents"`
	ChangePassword DBFunction `yaml:"change_password" json:"changePassword"`
	SetComment     DBFunction `yaml:"set_comment" json:"setComment"`
	SetAttribute   DBFunction `yaml:"set_attribute" json:"setAttribute"`
}

type BatchSettings struct {
	MaxConcurrency int `yaml:"max_concurrency" json:"maxConcurrency"`
}

const (
	ThemeLight  = "light"
	ThemeDark   = "dark"
	ThemeSystem = "system"
)

const (
	CommentViewFields = "fields"
	CommentViewRaw    = "raw"
)

type UISettings struct {
	Theme string `yaml:"theme" json:"theme"` // light | dark | system
	// CommentDefaultView is the comment editor's preferred mode for an empty comment
	// (create role / a role with no comment): fields | raw. Content-bearing comments always
	// auto-detect (JSON -> fields, plain text -> raw).
	CommentDefaultView string `yaml:"comment_default_view,omitempty" json:"commentDefaultView"`
}

// NormalizeTheme returns a valid theme preference; unknown values become system.
func NormalizeTheme(theme string) string {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case ThemeLight:
		return ThemeLight
	case ThemeDark:
		return ThemeDark
	default:
		return ThemeSystem
	}
}

// NormalizeCommentView returns a valid comment-view preference; unknown values become fields.
func NormalizeCommentView(view string) string {
	if strings.ToLower(strings.TrimSpace(view)) == CommentViewRaw {
		return CommentViewRaw
	}
	return CommentViewFields
}

type Config struct {
	Version     int           `yaml:"version" json:"version"`
	Categories  []Category    `yaml:"categories" json:"categories"`
	Clusters    []Cluster     `yaml:"clusters" json:"clusters"`
	DBFunctions DBFunctions   `yaml:"db_functions" json:"dbFunctions"`
	Batch       BatchSettings `yaml:"batch" json:"batch"`
	UI          UISettings    `yaml:"ui" json:"ui"`
	// ParentRoles are preconfigured parent groups offered as pick-list choices when
	// granting privileges (Create role / Alter role), instead of typing role names.
	ParentRoles []string `yaml:"parent_roles,omitempty" json:"parentRoles"`
	// CommentFields are the JSON keys the app surfaces as labeled inputs when a role
	// comment holds JSON (Create role / Alter role). Ordered; keys not listed here are
	// still shown generically. Defaults to full_name/e_mail.
	CommentFields []CommentField `yaml:"comment_fields,omitempty" json:"commentFields"`
	// Targets is the last target selection on the Operations page (cluster groups and/or
	// specific clusters), remembered across re-renders and restarts. Empty = "all groups".
	Targets TargetSelection `yaml:"targets,omitempty" json:"targets"`
}

// TargetSelection remembers the Operations-page target selection. An empty selection
// (both lists empty/absent) is treated by the UI as the default "all groups selected".
type TargetSelection struct {
	CategoryIDs []string `yaml:"category_ids,omitempty" json:"categoryIds"`
	ClusterIDs  []string `yaml:"cluster_ids,omitempty" json:"clusterIds"`
}

// CommentField maps a JSON comment key to a human label shown in the role form.
type CommentField struct {
	Key   string `yaml:"key" json:"key"`
	Label string `yaml:"label" json:"label"`
}

// ClustersConfig is the whole clusters+categories set, saved atomically from the staged
// Clusters editor. A cluster/category with an empty ID is treated as new.
type ClustersConfig struct {
	Clusters   []Cluster  `json:"clusters"`
	Categories []Category `json:"categories"`
}

type ClusterInput struct {
	Alias       string `json:"alias"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	Category    string `json:"category"`
	SSLMode     string `json:"sslMode"`
	ConnectUser string `json:"connectUser"`
}

type AuthContext struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type CreateRoleParams struct {
	LoginName  string `json:"loginName"`
	FullName   string `json:"fullName"`
	Email      string `json:"email"`
	ParentRole string `json:"parentRole"`
}

type RemoveRoleParams struct {
	LoginName string `json:"loginName"`
}

type GrantParentsParams struct {
	LoginName   string `json:"loginName"`
	ParentRoles string `json:"parentRoles"`
}

type RevokeParentsParams struct {
	LoginName   string `json:"loginName"`
	ParentRoles string `json:"parentRoles"`
}

type ChangePasswordParams struct {
	LoginName   string `json:"loginName"`
	NewPassword string `json:"newPassword"`
}

type SetCommentParams struct {
	LoginName string `json:"loginName"`
	Comment   string `json:"comment"`
}

// SetAttributeParams sets one ALTER ROLE attribute keyword (e.g. SUPERUSER / NOSUPERUSER).
type SetAttributeParams struct {
	LoginName string `json:"loginName"`
	Attribute string `json:"attribute"`
}

// SetConfigParams sets one role GUC: ALTER ROLE <login> SET <name> = '<value>'.
type SetConfigParams struct {
	LoginName   string `json:"loginName"`
	ConfigName  string `json:"configName"`
	ConfigValue string `json:"configValue"`
}

// ResetConfigParams clears one role GUC: ALTER ROLE <login> RESET <name>.
type ResetConfigParams struct {
	LoginName  string `json:"loginName"`
	ConfigName string `json:"configName"`
}

// OperationSpec is a single operation and its parameters, with no cluster targeting.
// Exactly one of the *Params pointers is non-nil, matching Operation.
type OperationSpec struct {
	Operation      string                `json:"operation"`
	CreateRole     *CreateRoleParams     `json:"createRole,omitempty"`
	RemoveRole     *RemoveRoleParams     `json:"removeRole,omitempty"`
	GrantParents   *GrantParentsParams   `json:"grantParents,omitempty"`
	RevokeParents  *RevokeParentsParams  `json:"revokeParents,omitempty"`
	ChangePassword *ChangePasswordParams `json:"changePassword,omitempty"`
	SetComment     *SetCommentParams     `json:"setComment,omitempty"`
	SetAttribute   *SetAttributeParams   `json:"setAttribute,omitempty"`
	SetConfig      *SetConfigParams      `json:"setConfig,omitempty"`
	ResetConfig    *ResetConfigParams    `json:"resetConfig,omitempty"`
}

// RunRequest targets clusters with a single operation. OperationSpec is embedded so the JSON
// wire shape is unchanged (operation + param pointers promote to the top level).
type RunRequest struct {
	OperationSpec
	CategoryIDs       []string    `json:"categoryIds"`
	ClusterIDs        []string    `json:"clusterIds"`
	Auth              AuthContext `json:"auth"`
	ConfirmProduction bool        `json:"confirmProduction"`
}

// ClusterOps is one cluster's ordered operation list (e.g. create_role, then
// grants/attributes/configs/comment), run as a single transaction on that cluster.
type ClusterOps struct {
	ClusterID  string          `json:"clusterId"`
	Operations []OperationSpec `json:"operations"`
}

// RoleBatchRequest applies, per cluster, an ordered list of operations inside one transaction.
type RoleBatchRequest struct {
	Clusters          []ClusterOps `json:"clusters"`
	Auth              AuthContext  `json:"auth"`
	ConfirmProduction bool         `json:"confirmProduction"`
}

type ClusterResult struct {
	ClusterID  string `json:"clusterId"`
	Alias      string `json:"alias"`
	Host       string `json:"host"`
	Category   string `json:"category"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	DurationMs int64  `json:"durationMs"`
	// Queries are the SQL statements executed on this cluster, in order (including the failing
	// one on rollback). Display-only; function-mode ops have their bind params inlined.
	Queries []string `json:"queries,omitempty"`
}

// ClusterProgress is a live per-cluster event during a role batch run.
// Phase is "running" (work started for this cluster) or "done" (result ready).
type ClusterProgress struct {
	ClusterID  string `json:"clusterId"`
	Alias      string `json:"alias"`
	Host       string `json:"host"`
	Category   string `json:"category"`
	Phase      string   `json:"phase"`  // "running" | "done"
	Status     string   `json:"status"` // "" while running; "ok"/"error" when done
	Message    string   `json:"message"`
	DurationMs int64    `json:"durationMs"`
	Queries    []string `json:"queries,omitempty"` // executed SQL (set on the "done" event)
}

type EnvImport struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
}

type TestConnectionRequest struct {
	ClusterID string      `json:"clusterId"`
	Auth      AuthContext `json:"auth"`
}

// AppVersion is build metadata shown in the UI (not config file version).
type AppVersion struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// RoleSearchRequest searches roles by a substring matched against role name and
// comment across the selected clusters/categories.
type RoleSearchRequest struct {
	Term        string      `json:"term"`
	CategoryIDs []string    `json:"categoryIds"`
	ClusterIDs  []string    `json:"clusterIds"`
	Auth        AuthContext `json:"auth"`
}

// RoleDetailsRequest loads one login's per-cluster state across the selected
// clusters/categories.
type RoleDetailsRequest struct {
	LoginName   string      `json:"loginName"`
	CategoryIDs []string    `json:"categoryIds"`
	ClusterIDs  []string    `json:"clusterIds"`
	Auth        AuthContext `json:"auth"`
}

// RoleMatch is one role found on one cluster during a search.
type RoleMatch struct {
	ClusterID string `json:"clusterId"`
	Alias     string `json:"alias"`
	Host      string `json:"host"`
	Category  string `json:"category"`
	LoginName string `json:"loginName"`
	Comment   string `json:"comment"`
	FullName  string `json:"fullName"`
	Error     string `json:"error,omitempty"` // per-cluster connect/query error, if any
}

// ClusterRoleDetail is one login's state on one cluster (parents = direct memberships).
type ClusterRoleDetail struct {
	ClusterID  string            `json:"clusterId"`
	Alias      string            `json:"alias"`
	Host       string            `json:"host"`
	Category   string            `json:"category"`
	Exists     bool              `json:"exists"`
	Comment    string            `json:"comment"`
	FullName   string            `json:"fullName"`
	Parents    []string          `json:"parents"`
	Attributes map[string]bool   `json:"attributes"`
	Settings   map[string]string `json:"settings"`
	Error      string            `json:"error,omitempty"`
	// DurationMs/Queries mirror ClusterResult so the load reports through the same run-status chip.
	DurationMs int64    `json:"durationMs"`
	Queries    []string `json:"queries,omitempty"`
}
