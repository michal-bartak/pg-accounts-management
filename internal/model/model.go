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
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
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
}

type BatchSettings struct {
	MaxConcurrency int `yaml:"max_concurrency" json:"maxConcurrency"`
}

type Config struct {
	Version     int           `yaml:"version" json:"version"`
	Categories  []Category    `yaml:"categories" json:"categories"`
	Clusters    []Cluster     `yaml:"clusters" json:"clusters"`
	DBFunctions DBFunctions   `yaml:"db_functions" json:"dbFunctions"`
	Batch       BatchSettings `yaml:"batch" json:"batch"`
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

type RunRequest struct {
	Operation       string   `json:"operation"`
	CategoryIDs     []string `json:"categoryIds"`
	ClusterIDs      []string `json:"clusterIds"`
	Auth            AuthContext `json:"auth"`
	CreateRole      *CreateRoleParams     `json:"createRole,omitempty"`
	RemoveRole      *RemoveRoleParams     `json:"removeRole,omitempty"`
	GrantParents    *GrantParentsParams   `json:"grantParents,omitempty"`
	RevokeParents   *RevokeParentsParams  `json:"revokeParents,omitempty"`
	ChangePassword  *ChangePasswordParams `json:"changePassword,omitempty"`
	ConfirmProduction bool `json:"confirmProduction"`
}

type ClusterResult struct {
	ClusterID string `json:"clusterId"`
	Alias     string `json:"alias"`
	Host      string `json:"host"`
	Category  string `json:"category"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	DurationMs int64 `json:"durationMs"`
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
