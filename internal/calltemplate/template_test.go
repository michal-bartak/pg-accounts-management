package calltemplate

import (
	"strings"
	"testing"
)

func TestBuildQueryFromTemplate_arrayOrNullWithParent(t *testing.T) {
	call := "admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})"
	args := map[string]string{
		"loginname":   "jdoe",
		"fullname":    "John Doe",
		"email":       "j@example.com",
		"parent_role": "gr_parent",
	}

	q, vals, err := BuildQueryFromTemplate(call, args, "create_role")
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

func TestBuildQueryFromTemplate_arrayOrNullEmptyParent(t *testing.T) {
	call := "fn(ARRAY['gr_a', 'gr_b'] || ${parent_role})"
	q, vals, err := BuildQueryFromTemplate(call, map[string]string{"parent_role": ""}, "create_role")
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
	in := "fn(ARRAY[${parent_role}, 'gr_a', 'gr_b'])"
	out := normalizeTemplate(in)
	if !strings.Contains(out, "ARRAY['gr_a', 'gr_b'] || ${parent_role}") {
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
