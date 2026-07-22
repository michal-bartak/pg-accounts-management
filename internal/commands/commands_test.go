package commands

import (
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/calltemplate"
	"github.com/michalbartak/dbaccounts/internal/config"
	"github.com/michalbartak/dbaccounts/internal/model"
)

func testConfig() model.Config {
	return config.DefaultConfig()
}

func baseRunRequest(op string) model.RunRequest {
	return model.RunRequest{
		OperationSpec: model.OperationSpec{Operation: op},
		CategoryIDs:   []string{"uat"},
		Auth:          model.AuthContext{User: "admin"},
	}
}

func TestValidateRequest_createRole(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpCreateRole)
	req.CreateRole = &model.CreateRoleParams{LoginName: "jdoe"}

	if err := ValidateRequest(cfg, req); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	req.CreateRole.LoginName = "  "
	if err := ValidateRequest(cfg, req); err == nil {
		t.Fatal("expected error for empty login")
	}
}

func TestValidateRequest_removeRole(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpRemoveRole)
	req.RemoveRole = &model.RemoveRoleParams{LoginName: "jdoe"}

	if err := ValidateRequest(cfg, req); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	req.RemoveRole = nil
	if err := ValidateRequest(cfg, req); err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("got: %v", err)
	}
}

func TestValidateRequest_grantParents(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpGrantParents)
	req.GrantParents = &model.GrantParentsParams{
		LoginName:   "jdoe",
		ParentRoles: "gr_a,gr_b",
	}

	if err := ValidateRequest(cfg, req); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequest_revokeParents(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpRevokeParents)
	req.RevokeParents = &model.RevokeParentsParams{
		LoginName:   "jdoe",
		ParentRoles: "gr_a",
	}

	if err := ValidateRequest(cfg, req); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequest_changePassword(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpChangePassword)
	req.ChangePassword = &model.ChangePasswordParams{
		LoginName:   "jdoe",
		NewPassword: "secret",
	}

	if err := ValidateRequest(cfg, req); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequest_requiresTargets(t *testing.T) {
	cfg := testConfig()
	req := baseRunRequest(OpRemoveRole)
	req.CategoryIDs = nil
	req.RemoveRole = &model.RemoveRoleParams{LoginName: "x"}

	if err := ValidateRequest(cfg, req); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateOperation_setAttribute_multiKeyword(t *testing.T) {
	cfg := testConfig()
	ok := model.OperationSpec{
		Operation:    OpSetAttribute,
		SetAttribute: &model.SetAttributeParams{LoginName: "jdoe", Attribute: "NOSUPERUSER NOLOGIN"},
	}
	if err := ValidateOperation(cfg, ok); err != nil {
		t.Fatalf("multi-keyword attribute rejected: %v", err)
	}
	bad := model.OperationSpec{
		Operation:    OpSetAttribute,
		SetAttribute: &model.SetAttributeParams{LoginName: "jdoe", Attribute: "SUPERUSER BOGUS"},
	}
	if err := ValidateOperation(cfg, bad); err == nil {
		t.Fatal("expected an unknown keyword to be rejected")
	}
	empty := model.OperationSpec{
		Operation:    OpSetAttribute,
		SetAttribute: &model.SetAttributeParams{LoginName: "jdoe", Attribute: "   "},
	}
	if err := ValidateOperation(cfg, empty); err == nil {
		t.Fatal("expected empty attribute to be rejected")
	}
}

func TestValidateRoleBatch(t *testing.T) {
	cfg := testConfig()
	good := model.RoleBatchRequest{
		Clusters: []model.ClusterOps{
			{ClusterID: "c1", Operations: []model.OperationSpec{
				{Operation: OpCreateRole, CreateRole: &model.CreateRoleParams{LoginName: "jdoe"}},
				{Operation: OpSetComment, SetComment: &model.SetCommentParams{LoginName: "jdoe", Comment: "{}"}},
			}},
		},
	}
	if err := ValidateRoleBatch(cfg, good); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	if err := ValidateRoleBatch(cfg, model.RoleBatchRequest{}); err == nil {
		t.Fatal("expected empty cluster list to be rejected")
	}
	emptyOps := model.RoleBatchRequest{Clusters: []model.ClusterOps{{ClusterID: "c1"}}}
	if err := ValidateRoleBatch(cfg, emptyOps); err == nil {
		t.Fatal("expected a cluster with no operations to be rejected")
	}
	badOp := model.RoleBatchRequest{
		Clusters: []model.ClusterOps{{ClusterID: "c1", Operations: []model.OperationSpec{
			{Operation: OpCreateRole, CreateRole: &model.CreateRoleParams{LoginName: ""}},
		}}},
	}
	if err := ValidateRoleBatch(cfg, badOp); err == nil {
		t.Fatal("expected a bad operation to propagate")
	}
}

func TestBuildArgs_allOperations(t *testing.T) {
	cfg := testConfig()

	tests := []struct {
		op   string
		spec model.OperationSpec
		want map[string]string
	}{
		{
			op: OpCreateRole,
			spec: model.OperationSpec{
				Operation:  OpCreateRole,
				CreateRole: &model.CreateRoleParams{LoginName: "u1", FullName: "Name", Email: "e@x.com", ParentRole: "gr_p"},
			},
			want: map[string]string{
				"loginname": "u1", "fullname": "Name", "email": "e@x.com", "parent_role": "gr_p",
			},
		},
		{
			op:   OpRemoveRole,
			spec: model.OperationSpec{Operation: OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: "u1"}},
			want: map[string]string{"loginname": "u1", "rolename": "u1"},
		},
		{
			op:   OpGrantParents,
			spec: model.OperationSpec{Operation: OpGrantParents, GrantParents: &model.GrantParentsParams{LoginName: "u1", ParentRoles: "a,b"}},
			want: map[string]string{"loginname": "u1", "parent_roles": "a,b"},
		},
		{
			op:   OpRevokeParents,
			spec: model.OperationSpec{Operation: OpRevokeParents, RevokeParents: &model.RevokeParentsParams{LoginName: "u1", ParentRoles: "gr_x"}},
			want: map[string]string{"loginname": "u1", "parent_roles": "gr_x"},
		},
		{
			op:   OpChangePassword,
			spec: model.OperationSpec{Operation: OpChangePassword, ChangePassword: &model.ChangePasswordParams{LoginName: "u1", NewPassword: "pw"}},
			want: map[string]string{"loginname": "u1", "new_password": "pw"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			fn, args, err := BuildArgs(cfg, tc.spec)
			if err != nil {
				t.Fatal(err)
			}
			if fn.Call == "" {
				t.Fatal("empty call template")
			}
			for k, v := range tc.want {
				if args[k] != v {
					t.Fatalf("%s: got %q want %q", k, args[k], v)
				}
			}
		})
	}
}

// TestBuildQuery_allOperations verifies SQL from default templates + command args.
func TestBuildQuery_allOperations(t *testing.T) {
	cfg := testConfig()

	tests := []struct {
		name       string
		operation  string
		spec       model.OperationSpec
		wantSubstr []string
		minBinds   int
	}{
		{
			name:      "create_role",
			operation: OpCreateRole,
			spec: model.OperationSpec{
				Operation:  OpCreateRole,
				CreateRole: &model.CreateRoleParams{LoginName: "jdoe", FullName: "John", Email: "j@x.com", ParentRole: "gr_extra"},
			},
			wantSubstr: []string{
				"SELECT admin_access.create_role(",
				"NULL",
				"ARRAY['gr_personal_users', 'gr_personal_users_ldap']::text[]",
				"::text[]",
			},
			minBinds: 4,
		},
		{
			name:      "create_role_empty_parent",
			operation: OpCreateRole,
			spec: model.OperationSpec{
				Operation:  OpCreateRole,
				CreateRole: &model.CreateRoleParams{LoginName: "jdoe", FullName: "John", Email: "j@x.com", ParentRole: ""},
			},
			wantSubstr: []string{"|| NULL"},
			minBinds:   3,
		},
		{
			name:       "remove_role",
			operation:  OpRemoveRole,
			spec:       model.OperationSpec{Operation: OpRemoveRole, RemoveRole: &model.RemoveRoleParams{LoginName: "jdoe"}},
			wantSubstr: []string{"SELECT your_schema.remove_app_role($1)"},
			minBinds:   1,
		},
		{
			name:       "grant_parents",
			operation:  OpGrantParents,
			spec:       model.OperationSpec{Operation: OpGrantParents, GrantParents: &model.GrantParentsParams{LoginName: "jdoe", ParentRoles: "gr_a,gr_b"}},
			wantSubstr: []string{"SELECT your_schema.grant_role_parents($1, $2)"},
			minBinds:   2,
		},
		{
			name:       "change_password",
			operation:  OpChangePassword,
			spec:       model.OperationSpec{Operation: OpChangePassword, ChangePassword: &model.ChangePasswordParams{LoginName: "jdoe", NewPassword: "s3cret"}},
			wantSubstr: []string{"SELECT your_schema.change_role_password($1, $2)"},
			minBinds:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn, args, err := BuildArgs(cfg, tc.spec)
			if err != nil {
				t.Fatal(err)
			}
			q, vals, err := calltemplate.BuildQueryFromTemplate(fn.Call, args, tc.operation)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tc.wantSubstr {
				if !strings.Contains(q, s) {
					t.Fatalf("query missing %q:\n%s", s, q)
				}
			}
			if len(vals) < tc.minBinds {
				t.Fatalf("binds: got %d want >= %d: %v", len(vals), tc.minBinds, vals)
			}
			if strings.Contains(q, "${") {
				t.Fatalf("unresolved placeholders: %s", q)
			}
		})
	}
}

func TestBuild_removeRole_statementMode(t *testing.T) {
	cfg := testConfig()
	cfg.DBFunctions.RemoveRole = model.DBFunction{
		Call:      "DROP ROLE ${loginname}",
		Execution: model.ExecutionStatement,
	}
	fn, args, err := BuildArgs(cfg, model.OperationSpec{
		Operation:  OpRemoveRole,
		RemoveRole: &model.RemoveRoleParams{LoginName: "jdoe"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sql, vals, useQuery, err := calltemplate.Build(fn.Call, args, OpRemoveRole, fn.Execution)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery || sql != "DROP ROLE jdoe" || len(vals) != 0 {
		t.Fatalf("useQuery=%v sql=%q vals=%v", useQuery, sql, vals)
	}
}

func TestBuild_revokeParents_statementMode(t *testing.T) {
	cfg := testConfig()
	fn, args, err := BuildArgs(cfg, model.OperationSpec{
		Operation:     OpRevokeParents,
		RevokeParents: &model.RevokeParentsParams{LoginName: "test", ParentRoles: "Gr_devs_all_ro"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sql, vals, useQuery, err := calltemplate.Build(fn.Call, args, OpRevokeParents, fn.Execution)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery || sql != "REVOKE Gr_devs_all_ro FROM test" || len(vals) != 0 {
		t.Fatalf("useQuery=%v sql=%q vals=%v", useQuery, sql, vals)
	}
}

func TestBuild_grantParents_statementMode(t *testing.T) {
	cfg := testConfig()
	cfg.DBFunctions.GrantParents = model.DBFunction{
		Call:      "GRANT ${parent_roles} TO ${loginname}",
		Execution: model.ExecutionStatement,
	}
	fn, args, err := BuildArgs(cfg, model.OperationSpec{
		Operation:    OpGrantParents,
		GrantParents: &model.GrantParentsParams{LoginName: "test", ParentRoles: "Gr_devs_all_ro"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sql, vals, useQuery, err := calltemplate.Build(fn.Call, args, OpGrantParents, fn.Execution)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery || sql != "GRANT Gr_devs_all_ro TO test" || len(vals) != 0 {
		t.Fatalf("useQuery=%v sql=%q vals=%v", useQuery, sql, vals)
	}
}

func TestBuildQuery_removeRole_customTemplate(t *testing.T) {
	cfg := testConfig()
	cfg.DBFunctions.RemoveRole = model.DBFunction{
		Call: "admin_access.drop_user(${loginname})",
	}
	fn, args, err := BuildArgs(cfg, model.OperationSpec{
		Operation:  OpRemoveRole,
		RemoveRole: &model.RemoveRoleParams{LoginName: "testuser"},
	})
	if err != nil {
		t.Fatal(err)
	}
	q, vals, err := calltemplate.BuildQueryFromTemplate(fn.Call, args, OpRemoveRole)
	if err != nil {
		t.Fatal(err)
	}
	if q != `SELECT admin_access.drop_user($1)` {
		t.Fatalf("got: %s", q)
	}
	if len(vals) != 1 || vals[0] != "testuser" {
		t.Fatalf("vals: %v", vals)
	}
}

func TestValidateRequest_setConfig(t *testing.T) {
	cfg := testConfig()
	ok := model.RunRequest{
		OperationSpec: model.OperationSpec{
			Operation: OpSetConfig,
			SetConfig: &model.SetConfigParams{LoginName: "t", ConfigName: "log_statement", ConfigValue: "all"},
		},
		ClusterIDs: []string{"c"},
	}
	if err := ValidateRequest(cfg, ok); err != nil {
		t.Fatalf("valid set_config rejected: %v", err)
	}
	bad := model.RunRequest{
		OperationSpec: model.OperationSpec{
			Operation: OpSetConfig,
			SetConfig: &model.SetConfigParams{LoginName: "t", ConfigName: "bad name;", ConfigValue: "x"},
		},
		ClusterIDs: []string{"c"},
	}
	if err := ValidateRequest(cfg, bad); err == nil {
		t.Fatal("expected invalid setting name to be rejected")
	}
	if !ValidConfigName("auto_explain.log_min_duration") || ValidConfigName("a b") {
		t.Fatal("ValidConfigName wrong")
	}
	// set_config/reset_config build empty DBFunction (handled directly in pg).
	fn, args, err := BuildArgs(cfg, ok.OperationSpec)
	if err != nil || fn.Call != "" || args["config_name"] != "log_statement" || args["config_value"] != "all" {
		t.Fatalf("BuildArgs set_config: fn=%q args=%v err=%v", fn.Call, args, err)
	}
}

func TestRequiresProductionConfirm(t *testing.T) {
	cats := []model.Category{
		{ID: "uat", Confirm: false},
		{ID: "production", Confirm: true},
	}
	clusters := []model.Cluster{
		{ID: "1", Category: "uat"},
		{ID: "2", Category: "production"},
	}
	if !RequiresProductionConfirm(cats, clusters) {
		t.Fatal("expected true when a target's group has Confirm=true")
	}
	if RequiresProductionConfirm(cats, []model.Cluster{{Category: "uat"}}) {
		t.Fatal("expected false when no target's group requires confirm")
	}
	// A group without the flag never triggers, even if named 'production'.
	noFlag := []model.Category{{ID: "production", Confirm: false}}
	if RequiresProductionConfirm(noFlag, []model.Cluster{{Category: "production"}}) {
		t.Fatal("expected false when the group's Confirm flag is off")
	}
}
