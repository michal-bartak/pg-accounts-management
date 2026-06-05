package config

import (
	"fmt"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/calltemplate"
	"github.com/michalbartak/dbaccounts/internal/model"
)

func migrateDBFunctions(fns *model.DBFunctions) {
	defaults := DefaultConfig().DBFunctions
	fns.CreateRole = migrateOne(fns.CreateRole, defaults.CreateRole, "create_role")
	fns.RemoveRole = migrateOne(fns.RemoveRole, defaults.RemoveRole, "remove_role")
	fns.GrantParents = migrateOne(fns.GrantParents, defaults.GrantParents, "grant_parents")
	fns.RevokeParents = migrateOne(fns.RevokeParents, defaults.RevokeParents, "revoke_parents")
	fns.ChangePassword = migrateOne(fns.ChangePassword, defaults.ChangePassword, "change_password")
}

func normalizeDBFunction(fn model.DBFunction) model.DBFunction {
	fn.Execution = model.NormalizeExecution(fn.Execution)
	return fn
}

func migrateOne(fn, def model.DBFunction, operation string) model.DBFunction {
	call := strings.TrimSpace(fn.Call)
	if call != "" {
		if needsCreateRoleTemplateFix(operation, call) {
			return def
		}
		return normalizeDBFunction(fn)
	}
	// Broken config: full SQL stored in name field.
	if strings.Contains(fn.Name, "(") {
		return def
	}
	if fn.Name != "" && len(fn.Params) > 0 {
		parts := make([]string, 0, len(fn.Params))
		for _, p := range fn.Params {
			p = strings.TrimSpace(p)
			switch {
			case p == "@null" || strings.EqualFold(p, "NULL"):
				parts = append(parts, "NULL")
			case strings.HasPrefix(p, "@array:"):
				// Cannot reliably migrate array specs; use default template.
				return def
			default:
				parts = append(parts, "${"+p+"}")
			}
		}
		fn.Call = fn.Name + "(" + strings.Join(parts, ", ") + ")"
		return normalizeDBFunction(fn)
	}
	if fn.Name != "" {
		fn.Call = fn.Name + "()"
		return normalizeDBFunction(fn)
	}
	return def
}

// needsCreateRoleTemplateFix detects invalid ${Array[...]} style templates from older UI attempts.
func needsCreateRoleTemplateFix(operation, call string) bool {
	if operation != "create_role" {
		return false
	}
	lower := strings.ToLower(call)
	return strings.Contains(call, "${Array") ||
		strings.Contains(call, "array_concat:") ||
		strings.Contains(lower, "array['parent_role'") ||
		(strings.Contains(call, "ARRAY[${parent_role}") && !strings.Contains(call, "|| ${parent_role}"))
}

func validateDBFunctions(fns model.DBFunctions) error {
	checks := []struct {
		op string
		fn model.DBFunction
	}{
		{"create_role", fns.CreateRole},
		{"remove_role", fns.RemoveRole},
		{"grant_parents", fns.GrantParents},
		{"revoke_parents", fns.RevokeParents},
		{"change_password", fns.ChangePassword},
	}
	for _, c := range checks {
		exec := model.NormalizeExecution(c.fn.Execution)
		if err := calltemplate.ValidateCallTemplateWithExecution(c.fn.Call, c.op, exec); err != nil {
			return fmt.Errorf("%s: %w", c.op, err)
		}
	}
	return nil
}
