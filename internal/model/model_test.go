package model

import (
	"encoding/json"
	"testing"
)

// TestRunRequest_wireShapeFlattened verifies that embedding OperationSpec keeps RunRequest's JSON
// shape flat (operation + param pointers at the top level), so the existing single-op wire format
// is unchanged.
func TestRunRequest_wireShapeFlattened(t *testing.T) {
	in := []byte(`{"operation":"create_role","clusterIds":["c1"],"confirmProduction":true,` +
		`"createRole":{"loginName":"jdoe","fullName":"John"}}`)
	var req RunRequest
	if err := json.Unmarshal(in, &req); err != nil {
		t.Fatal(err)
	}
	if req.Operation != "create_role" {
		t.Fatalf("operation not promoted: %q", req.Operation)
	}
	if req.CreateRole == nil || req.CreateRole.LoginName != "jdoe" {
		t.Fatalf("createRole not promoted: %+v", req.CreateRole)
	}
	if len(req.ClusterIDs) != 1 || !req.ConfirmProduction {
		t.Fatalf("targeting fields wrong: %+v", req)
	}

	// Marshal back: operation + createRole must remain top-level (not nested under a key).
	out, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !contains(s, `"operation":"create_role"`) || !contains(s, `"createRole":`) {
		t.Fatalf("flattened shape lost: %s", s)
	}
	if contains(s, `"OperationSpec"`) || contains(s, `"operationSpec"`) {
		t.Fatalf("embedded struct leaked a key: %s", s)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
