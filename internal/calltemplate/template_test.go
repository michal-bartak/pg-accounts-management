package calltemplate

import (
	"strings"
	"testing"
)

func TestBuildQueryFromTemplate_arrayOrNullWithParent(t *testing.T) {
	call := "admin_access.create_role(${loginname}, NULL, ${full_name}, ${e_mail}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_roles})"
	args := map[string]string{
		"loginname":    "jdoe",
		"full_name":    `"John Doe"`,
		"e_mail":       `"j@example.com"`,
		"parent_roles": "gr_parent",
	}

	q, vals, err := BuildQueryFromTemplate(call, args, "create_role", "full_name", "e_mail")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "ARRAY['gr_personal_users', 'gr_personal_users_ldap']::text[] || $") {
		t.Fatalf("query: %s", q)
	}
	if !strings.Contains(q, "::text[]") {
		t.Fatalf("expected text[] bind for parent: %s", q)
	}
	var found bool
	for _, v := range vals {
		arr, ok := v.([]string)
		if ok && len(arr) == 1 && arr[0] == "gr_parent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("vals: %v", vals)
	}
}

func TestBuildQueryFromTemplate_arrayOrNullMultipleParents(t *testing.T) {
	call := "fn(ARRAY['gr_a'] || ${parent_roles})"
	q, vals, err := BuildQueryFromTemplate(call, map[string]string{"parent_roles": "gr_x, gr_y ,gr_z"}, "create_role")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "::text[]") {
		t.Fatalf("query: %s", q)
	}
	var got []string
	for _, v := range vals {
		if arr, ok := v.([]string); ok {
			got = arr
		}
	}
	if len(got) != 3 || got[0] != "gr_x" || got[1] != "gr_y" || got[2] != "gr_z" {
		t.Fatalf("expected three trimmed parents, got %v", got)
	}
}

func TestBuildQueryFromTemplate_arrayOrNullEmptyParent(t *testing.T) {
	call := "fn(ARRAY['gr_a', 'gr_b'] || ${parent_roles})"
	q, vals, err := BuildQueryFromTemplate(call, map[string]string{"parent_roles": ""}, "create_role")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "ARRAY['gr_a', 'gr_b']::text[] || NULL") {
		t.Fatalf("query: %s", q)
	}
	if len(vals) != 0 {
		t.Fatalf("expected no binds for empty parent, got %v", vals)
	}
}

func TestNormalizeTemplate_arrayLiteralForm(t *testing.T) {
	in := "fn(ARRAY[${parent_roles}, 'gr_a', 'gr_b'])"
	out := normalizeTemplate(in)
	if !strings.Contains(out, "ARRAY['gr_a', 'gr_b'] || ${parent_roles}") {
		t.Fatalf("got: %s", out)
	}
}

func TestValidateCallTemplate_rejectsInvalidArrayPlaceholder(t *testing.T) {
	call := "fn(${loginname}, ${Array['parent_role']})"
	err := ValidateCallTemplate(call, "create_role")
	if err == nil {
		t.Fatal("expected error for ${Array...}")
	}
}

func TestValidateCallTemplate_rejectsSelect(t *testing.T) {
	err := ValidateCallTemplate("SELECT fn(${loginname})", "remove_role")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateCallTemplate_removeRoleWhitelist(t *testing.T) {
	err := ValidateCallTemplate("app.remove(${loginname})", "remove_role")
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateCallTemplate("app.remove(${fullname})", "remove_role")
	if err == nil {
		t.Fatal("fullname not allowed for remove_role")
	}
}

func TestBuildArrayConcatValue(t *testing.T) {
	got := buildArrayConcatValue("parent", []string{"a", "b"})
	if len(got) != 3 || got[0] != "parent" || got[2] != "b" {
		t.Fatalf("%v", got)
	}
	got = buildArrayConcatValue("", []string{"a", "b"})
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("%v", got)
	}
}
