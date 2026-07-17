package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/model"
)

const (
	OpCreateRole     = "create_role"
	OpRemoveRole     = "remove_role"
	OpGrantParents   = "grant_parents"
	OpRevokeParents  = "revoke_parents"
	OpChangePassword = "change_password"
	OpSetComment     = "set_comment"
	OpSetAttribute   = "set_attribute"
	OpSetConfig      = "set_config"
	OpResetConfig    = "reset_config"
)

// configNameRE validates a GUC name (optionally namespaced, e.g. auto_explain.log_min_duration).
var configNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// ValidConfigName reports whether name is an acceptable role GUC name.
func ValidConfigName(name string) bool {
	return configNameRE.MatchString(strings.TrimSpace(name))
}

// allowedAttributeKeywords are the ALTER ROLE attribute keywords the app may emit.
var allowedAttributeKeywords = map[string]bool{
	"SUPERUSER": true, "NOSUPERUSER": true,
	"CREATEROLE": true, "NOCREATEROLE": true,
	"CREATEDB": true, "NOCREATEDB": true,
	"INHERIT": true, "NOINHERIT": true,
	"LOGIN": true, "NOLOGIN": true,
	"REPLICATION": true, "NOREPLICATION": true,
	"BYPASSRLS": true, "NOBYPASSRLS": true,
}

func BuildArgs(cfg model.Config, req model.RunRequest) (model.DBFunction, map[string]string, error) {
	switch req.Operation {
	case OpCreateRole:
		if req.CreateRole == nil {
			return model.DBFunction{}, nil, fmt.Errorf("create role parameters missing")
		}
		p := req.CreateRole
		return cfg.DBFunctions.CreateRole, map[string]string{
			"loginname":   p.LoginName,
			"fullname":    p.FullName,
			"email":       p.Email,
			"parent_role": p.ParentRole,
		}, nil
	case OpRemoveRole:
		if req.RemoveRole == nil {
			return model.DBFunction{}, nil, fmt.Errorf("remove role parameters missing")
		}
		login := req.RemoveRole.LoginName
		return cfg.DBFunctions.RemoveRole, map[string]string{
			"loginname": login,
			"rolename":  login,
		}, nil
	case OpGrantParents:
		if req.GrantParents == nil {
			return model.DBFunction{}, nil, fmt.Errorf("grant parents parameters missing")
		}
		return cfg.DBFunctions.GrantParents, map[string]string{
			"loginname":    req.GrantParents.LoginName,
			"parent_roles": req.GrantParents.ParentRoles,
		}, nil
	case OpRevokeParents:
		if req.RevokeParents == nil {
			return model.DBFunction{}, nil, fmt.Errorf("revoke parents parameters missing")
		}
		return cfg.DBFunctions.RevokeParents, map[string]string{
			"loginname":    req.RevokeParents.LoginName,
			"parent_roles": req.RevokeParents.ParentRoles,
		}, nil
	case OpChangePassword:
		if req.ChangePassword == nil {
			return model.DBFunction{}, nil, fmt.Errorf("change password parameters missing")
		}
		return cfg.DBFunctions.ChangePassword, map[string]string{
			"loginname":    req.ChangePassword.LoginName,
			"new_password": req.ChangePassword.NewPassword,
		}, nil
	case OpSetComment:
		if req.SetComment == nil {
			return model.DBFunction{}, nil, fmt.Errorf("set comment parameters missing")
		}
		return cfg.DBFunctions.SetComment, map[string]string{
			"loginname": req.SetComment.LoginName,
			"comment":   req.SetComment.Comment,
		}, nil
	case OpSetAttribute:
		if req.SetAttribute == nil {
			return model.DBFunction{}, nil, fmt.Errorf("set attribute parameters missing")
		}
		return cfg.DBFunctions.SetAttribute, map[string]string{
			"loginname": req.SetAttribute.LoginName,
			"attribute": req.SetAttribute.Attribute,
		}, nil
	case OpSetConfig:
		if req.SetConfig == nil {
			return model.DBFunction{}, nil, fmt.Errorf("set config parameters missing")
		}
		// Built directly in pg.ExecuteOperation (no template); empty DBFunction.
		return model.DBFunction{}, map[string]string{
			"loginname":    req.SetConfig.LoginName,
			"config_name":  req.SetConfig.ConfigName,
			"config_value": req.SetConfig.ConfigValue,
		}, nil
	case OpResetConfig:
		if req.ResetConfig == nil {
			return model.DBFunction{}, nil, fmt.Errorf("reset config parameters missing")
		}
		return model.DBFunction{}, map[string]string{
			"loginname":   req.ResetConfig.LoginName,
			"config_name": req.ResetConfig.ConfigName,
		}, nil
	default:
		return model.DBFunction{}, nil, fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func ValidateRequest(cfg model.Config, req model.RunRequest) error {
	if req.Operation == "" {
		return fmt.Errorf("operation is required")
	}
	if len(req.CategoryIDs) == 0 && len(req.ClusterIDs) == 0 {
		return fmt.Errorf("select at least one category or cluster")
	}
	switch req.Operation {
	case OpCreateRole:
		if req.CreateRole == nil || strings.TrimSpace(req.CreateRole.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
		// parent_role optional when using ${array_concat:parent_role,...} (empty → fixed groups only)
	case OpRemoveRole:
		if req.RemoveRole == nil || strings.TrimSpace(req.RemoveRole.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
	case OpGrantParents:
		if req.GrantParents == nil || strings.TrimSpace(req.GrantParents.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
	case OpRevokeParents:
		if req.RevokeParents == nil || strings.TrimSpace(req.RevokeParents.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
	case OpChangePassword:
		if req.ChangePassword == nil || strings.TrimSpace(req.ChangePassword.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
	case OpSetComment:
		if req.SetComment == nil || strings.TrimSpace(req.SetComment.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
	case OpSetAttribute:
		if req.SetAttribute == nil || strings.TrimSpace(req.SetAttribute.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
		if !allowedAttributeKeywords[strings.ToUpper(strings.TrimSpace(req.SetAttribute.Attribute))] {
			return fmt.Errorf("unsupported role attribute: %q", req.SetAttribute.Attribute)
		}
	case OpSetConfig:
		if req.SetConfig == nil || strings.TrimSpace(req.SetConfig.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
		if !ValidConfigName(req.SetConfig.ConfigName) {
			return fmt.Errorf("invalid setting name: %q", req.SetConfig.ConfigName)
		}
	case OpResetConfig:
		if req.ResetConfig == nil || strings.TrimSpace(req.ResetConfig.LoginName) == "" {
			return fmt.Errorf("login name is required")
		}
		if !ValidConfigName(req.ResetConfig.ConfigName) {
			return fmt.Errorf("invalid setting name: %q", req.ResetConfig.ConfigName)
		}
	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
	return nil
}

// RequiresProductionConfirm reports whether any target cluster belongs to a group
// flagged Confirm (the per-group replacement for the old hardcoded "production" id).
func RequiresProductionConfirm(categories []model.Category, clusters []model.Cluster) bool {
	confirm := make(map[string]bool, len(categories))
	for _, cat := range categories {
		confirm[cat.ID] = cat.Confirm
	}
	for _, c := range clusters {
		if confirm[c.Category] {
			return true
		}
	}
	return false
}
