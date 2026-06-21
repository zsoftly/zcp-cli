package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// execCapture runs a command while capturing os.Stdout (where the output.Printer
// writes tables and success lines), returning that text plus cobra's stderr. The
// printer is constructed from os.Stdout inside the command run, so swapping the
// pipe in before execCmd captures it.
func execCapture(t *testing.T, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	old := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	cobraOut, stderr, err := execCmd(t, cmd, args...)
	wp.Close()
	os.Stdout = old
	piped, _ := io.ReadAll(rp)
	// Commands use two output sinks: the output.Printer writes tables/success
	// lines to os.Stdout, while detail views write to cobra's OutOrStdout buffer.
	// Combine both so assertions see all command output.
	return cobraOut + string(piped), stderr, err
}

// rbacServer mocks the /users, /roles, /permissions routes enough to drive the
// sub-user, role, and permission commands end-to-end without a real API.
func rbacServer(t *testing.T, captured *map[string]interface{}, lastMethodPath *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lastMethodPath != nil {
			*lastMethodPath = r.Method + " " + r.URL.Path
		}
		if captured != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
			body := map[string]interface{}{}
			json.NewDecoder(r.Body).Decode(&body)
			*captured = body
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/permissions":
			io.WriteString(w, `{"status":"Success","data":[
				{"id":"p1","name":"DNS Read","slug":"dns-read","category":"DNS","status":true},
				{"id":"p2","name":"Virtual Machine Read","slug":"virtual-machine-read","category":"Virtual Machine","status":true}]}`)
		case r.URL.Path == "/roles" && r.Method == http.MethodGet:
			io.WriteString(w, `{"status":"Success","data":[
				{"id":"r1","name":"Owner","slug":"owner","users":[{"id":"u0","name":"O","email":"o@z.ca"}]},
				{"id":"r2","name":"Custom","slug":"custom","users":[]}]}`)
		case r.URL.Path == "/roles" && r.Method == http.MethodPost:
			io.WriteString(w, `{"status":"Success","data":{"id":"r9","name":"Custom","slug":"custom"}}`)
		case r.URL.Path == "/roles/custom" && r.Method == http.MethodGet:
			io.WriteString(w, `{"status":"Success","data":{"id":"r2","name":"Custom","slug":"custom","description":"old","permissions":[{"slug":"dns-read","name":"DNS Read","category":"DNS"}],"users":[]}}`)
		case r.URL.Path == "/roles/custom" && r.Method == http.MethodPut:
			io.WriteString(w, `{"status":"Success","data":{"id":"r2","slug":"custom"}}`)
		case r.URL.Path == "/roles/custom" && r.Method == http.MethodDelete:
			io.WriteString(w, `{"status":"Success","message":"Role deleted successfully."}`)
		case r.URL.Path == "/users" && r.Method == http.MethodGet:
			io.WriteString(w, `{"status":"Success","data":[
				{"id":"u1","name":"Jane","email":"jane@z.ca","user_type":"sub_user","is_blocked":false,"user_status":"Active","role":{"slug":"service-viewer"},"projects":[{"slug":"default-9"}]}]}`)
		case r.URL.Path == "/users" && r.Method == http.MethodPost:
			io.WriteString(w, `{"status":"Success","data":{"id":"u9","name":"New","email":"new@z.ca"}}`)
		case r.URL.Path == "/users/u1" && r.Method == http.MethodPut:
			io.WriteString(w, `{"status":"Success","data":{"id":"u1","is_blocked":true}}`)
		case r.URL.Path == "/users/u1" && r.Method == http.MethodDelete:
			io.WriteString(w, `{"status":"Success","message":"deleted"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"status":"Error","message":"not found"}`)
		}
	}))
}

func setRBACToken(t *testing.T) {
	t.Helper()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	t.Cleanup(func() { os.Unsetenv("ZCP_BEARER_TOKEN") })
}

func TestPermissionList(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	out, _, err := execCapture(t, NewPermissionCmd(), "list", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("permission list error: %v", err)
	}
	if !strings.Contains(out, "dns-read") || !strings.Contains(out, "DNS") {
		t.Errorf("output missing permission rows:\n%s", out)
	}
}

func TestPermissionListCategoryFilter(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	out, _, err := execCapture(t, NewPermissionCmd(), "list", "--category", "DNS", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(out, "dns-read") || strings.Contains(out, "virtual-machine-read") {
		t.Errorf("category filter failed:\n%s", out)
	}
}

func TestRoleList(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	out, _, err := execCapture(t, NewRoleCmd(), "list", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("role list error: %v", err)
	}
	if !strings.Contains(out, "owner") || !strings.Contains(out, "yes") {
		t.Errorf("role list missing predefined marker:\n%s", out)
	}
}

func TestRoleGetShowsPermissions(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	out, _, err := execCapture(t, NewRoleCmd(), "get", "custom", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("role get error: %v", err)
	}
	if !strings.Contains(out, "dns-read") {
		t.Errorf("role get missing permissions:\n%s", out)
	}
}

func TestRoleCreateRequiresPermission(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewRoleCmd(), "create", "--name", "X", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "permission") {
		t.Errorf("expected missing-permission error, got %v", err)
	}
}

func TestRoleCreateSendsPermissions(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	srv := rbacServer(t, &body, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewRoleCmd(), "create", "--name", "Custom",
		"--permission", "dns-read", "--permission", "virtual-machine-read", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("role create error: %v", err)
	}
	if perms, _ := body["permissions"].([]interface{}); len(perms) != 2 {
		t.Errorf("expected 2 permissions sent, got %v", body["permissions"])
	}
}

func TestRoleUpdatePreservesPermissionsWhenOnlyDescChanges(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	srv := rbacServer(t, &body, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewRoleCmd(), "update", "custom", "--description", "new", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("role update error: %v", err)
	}
	// description changed but the current permissions (dns-read) must be echoed back.
	if body["description"] != "new" {
		t.Errorf("description not sent: %v", body["description"])
	}
	if perms, _ := body["permissions"].([]interface{}); len(perms) != 1 || perms[0] != "dns-read" {
		t.Errorf("permissions not preserved on desc-only update: %v", body["permissions"])
	}
}

func TestRoleUpdateCanClearDescription(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	srv := rbacServer(t, &body, nil)
	defer srv.Close()

	// An explicit empty --description must be SENT (the API clears on "" and
	// preserves when the field is absent), so it must not be dropped by omitempty.
	_, _, err := execCmd(t, NewRoleCmd(), "update", "custom", "--description", "", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("role update error: %v", err)
	}
	desc, ok := body["description"]
	if !ok {
		t.Fatal("description was omitted from the request; an explicit empty value must be sent to clear it")
	}
	if desc != "" {
		t.Errorf("description = %v, want empty string", desc)
	}
}

func TestRoleUpdatePredefinedRejected(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewRoleCmd(), "update", "owner", "--description", "x", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "predefined") {
		t.Errorf("expected predefined rejection, got %v", err)
	}
}

func TestRoleDeletePredefinedRejected(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewRoleCmd(), "delete", "service-viewer", "--yes", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "predefined") {
		t.Errorf("expected predefined rejection, got %v", err)
	}
}

func TestSubUserList(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	out, _, err := execCapture(t, NewSubUserCmd(), "list", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("sub-user list error: %v", err)
	}
	if !strings.Contains(out, "jane@z.ca") || !strings.Contains(out, "service-viewer") {
		t.Errorf("sub-user list output:\n%s", out)
	}
}

func TestSubUserCreateSendsRequiredFields(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	srv := rbacServer(t, &body, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewSubUserCmd(), "create", "--name", "New", "--email", "new@z.ca",
		"--password", "Abc12345!", "--role", "service-viewer", "--project", "default-9", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("sub-user create error: %v", err)
	}
	if body["is_user_password"] != true || body["auth_user"] != "customer" {
		t.Errorf("create body missing defaults: %+v", body)
	}
	if projs, _ := body["projects"].([]interface{}); len(projs) != 1 {
		t.Errorf("projects not sent: %v", body["projects"])
	}
}

func TestSubUserUpdateResolvesByEmailAndPreservesProjects(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	var mp string
	srv := rbacServer(t, &body, &mp)
	defer srv.Close()

	_, _, err := execCmd(t, NewSubUserCmd(), "update", "jane@z.ca", "--role", "service-administrator", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("sub-user update error: %v", err)
	}
	if mp != "PUT /users/u1" {
		t.Errorf("expected PUT /users/u1, got %q", mp)
	}
	if body["role"] != "service-administrator" {
		t.Errorf("role not changed: %v", body["role"])
	}
	// email + projects must be echoed back from the current record (API requires them).
	if body["email"] != "jane@z.ca" {
		t.Errorf("email not preserved: %v", body["email"])
	}
	if projs, _ := body["projects"].([]interface{}); len(projs) != 1 || projs[0] != "default-9" {
		t.Errorf("projects not preserved: %v", body["projects"])
	}
}

func TestSubUserBlockSetsIsBlocked(t *testing.T) {
	setRBACToken(t)
	var body map[string]interface{}
	srv := rbacServer(t, &body, nil)
	defer srv.Close()

	_, _, err := execCmd(t, NewSubUserCmd(), "block", "jane@z.ca", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("sub-user block error: %v", err)
	}
	if body["is_blocked"] != true {
		t.Errorf("is_blocked not set true: %v", body["is_blocked"])
	}
}

func TestSubUserDeleteIdempotentWhenMissing(t *testing.T) {
	setRBACToken(t)
	srv := rbacServer(t, nil, nil)
	defer srv.Close()

	_, stderr, err := execCmd(t, NewSubUserCmd(), "delete", "ghost@z.ca", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected idempotent success, got %v", err)
	}
	if !strings.Contains(stderr, "already deleted") {
		t.Errorf("expected already-deleted message, got %q", stderr)
	}
}

func TestSubUserDeletePropagatesListError(t *testing.T) {
	setRBACToken(t)
	// The list lookup fails (e.g. expired token). Delete must surface the error,
	// NOT report a successful/idempotent deletion. 401 is non-retryable, so the
	// test stays fast.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"status":"Error","message":"token expired"}`)
	}))
	defer srv.Close()

	_, stderr, err := execCmd(t, NewSubUserCmd(), "delete", "jane@z.ca", "--yes", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected error when the user lookup fails, got nil")
	}
	if strings.Contains(stderr, "already deleted") {
		t.Errorf("a lookup failure must not be reported as a deletion; stderr=%q", stderr)
	}
}
