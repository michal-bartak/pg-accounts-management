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
		Operation:   op,
		CategoryIDs: []string{"uat"},
		ClusterIDs:  nil,
		Auth:        model.AuthContext{User: "admin"},
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

func TestBuildArgs_allOperations(t *testing.T) {
	cfg := testConfig()

	tests := []struct {
		op   string
		req  model.RunRequest
		want map[string]string
	}{
		{
			op: OpCreateRole,
			req: func() model.RunRequest {
				r := baseRunRequest(OpCreateRole)
				r.CreateRole = &model.CreateRoleParams{
					LoginName: "u1", FullName: "Name", Email: "e@x.com", ParentRole: "gr_p",
				}
				return r
			}(),
			want: map[string]string{
				"loginname": "u1", "fullname": "Name", "email": "e@x.com", "parent_role": "gr_p",
			},
		},
		{
			op: OpRemoveRole,
			req: func() model.RunRequest {
				r := baseRunRequest(OpRemoveRole)
				r.RemoveRole = &model.RemoveRoleParams{LoginName: "u1"}
				return r
			}(),
			want: map[string]string{"loginname": "u1", "rolename": "u1"},
		},
		{
			op: OpGrantParents,
			req: func() model.RunRequest {
				r := baseRunRequest(OpGrantParents)
				r.GrantParents = &model.GrantParentsParams{LoginName: "u1", ParentRoles: "a,b"}
				return r
			}(),
			want: map[string]string{"loginname": "u1", "parent_roles": "a,b"},
		},
		{
			op: OpRevokeParents,
			req: func() model.RunRequest {
				r := baseRunRequest(OpRevokeParents)
				r.RevokeParents = &model.RevokeParentsParams{LoginName: "u1", ParentRoles: "gr_x"}
				return r
			}(),
			want: map[string]string{"loginname": "u1", "parent_roles": "gr_x"},
		},
		{
			op: OpChangePassword,
			req: func() model.RunRequest {
				r := baseRunRequest(OpChangePassword)
				r.ChangePassword = &model.ChangePasswordParams{LoginName: "u1", NewPassword: "pw"}
				return r
			}(),
			want: map[string]string{"loginname": "u1", "new_password": "pw"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.op, func(t *testing.T) {
			fn, args, err := BuildArgs(cfg, tc.req)
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
		buildReq   func() model.RunRequest
		wantSubstr []string
		minBinds   int
	}{
		{
			name:      "create_role",
			operation: OpCreateRole,
			buildReq: func() model.RunRequest {
				return model.RunRequest{
					Operation: OpCreateRole,
					CreateRole: &model.CreateRoleParams{
						LoginName: "jdoe", FullName: "John", Email: "j@x.com", ParentRole: "gr_extra",
					},
				}
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
			buildReq: func() model.RunRequest {
				return model.RunRequest{
					Operation: OpCreateRole,
					CreateRole: &model.CreateRoleParams{
						LoginName: "jdoe", FullName: "John", Email: "j@x.com", ParentRole: "",
					},
				}
			},
			wantSubstr: []string{"|| NULL"},
			minBinds:   3,
		},
		{
			name:      "remove_role",
			operation: OpRemoveRole,
			buildReq: func() model.RunRequest {
				return model.RunRequest{
					Operation:  OpRemoveRole,
					RemoveRole: &model.RemoveRoleParams{LoginName: "jdoe"},
				}
			},
			wantSubstr: []string{"SELECT your_schema.remove_app_role($1)"},
			minBinds:   1,
		},
		{
			name:      "grant_parents",
			operation: OpGrantParents,
			buildReq: func() model.RunRequest {
				return model.RunRequest{
					Operation: OpGrantParents,
					GrantParents: &model.GrantParentsParams{
						LoginName: "jdoe", ParentRoles: "gr_a,gr_b",
					},
				}
			},
			wantSubstr: []string{"SELECT your_schema.grant_role_parents($1, $2)"},
			minBinds:   2,
		},
		{
			name:      "change_password",
			operation: OpChangePassword,
			buildReq: func() model.RunRequest {
				return model.RunRequest{
					Operation: OpChangePassword,
					ChangePassword: &model.ChangePasswordParams{
						LoginName: "jdoe", NewPassword: "s3cret",
					},
				}
			},
			wantSubstr: []string{"SELECT your_schema.change_role_password($1, $2)"},
			minBinds:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.buildReq()
			fn, args, err := BuildArgs(cfg, req)
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
	req := model.RunRequest{
		Operation:  OpRemoveRole,
		RemoveRole: &model.RemoveRoleParams{LoginName: "jdoe"},
	}
	fn, args, err := BuildArgs(cfg, req)
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
	fn, args, err := BuildArgs(cfg, model.RunRequest{
		Operation: OpRevokeParents,
		RevokeParents: &model.RevokeParentsParams{
			LoginName:   "test",
			ParentRoles: "Gr_devs_all_ro",
		},
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
	req := model.RunRequest{
		Operation: OpGrantParents,
		GrantParents: &model.GrantParentsParams{
			LoginName:   "test",
			ParentRoles: "Gr_devs_all_ro",
		},
	}
	fn, args, err := BuildArgs(cfg, req)
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
	req := model.RunRequest{
		Operation:  OpRemoveRole,
		RemoveRole: &model.RemoveRoleParams{LoginName: "testuser"},
	}
	fn, args, err := BuildArgs(cfg, req)
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

func TestRequiresProductionConfirm(t *testing.T) {
	clusters := []model.Cluster{
		{ID: "1", Category: "uat"},
		{ID: "2", Category: "production"},
	}
	if !RequiresProductionConfirm(nil, clusters) {
		t.Fatal("expected true")
	}
	if RequiresProductionConfirm(nil, []model.Cluster{{Category: "uat"}}) {
		t.Fatal("expected false")
	}
}
