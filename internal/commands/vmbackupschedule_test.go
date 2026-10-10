package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVMBackupScheduleCreateSendsPortalContract(t *testing.T) {
	var got map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":[{"slug":"vm-1"}]}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/scheduler-actions" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(w, `{"data":{"id":"policy-1","name":"nightly","slug":"vm-1","interval":"weeklyOn","day":0,"at":"13:00","timezone":"America/Toronto","retention_policy":2,"status":"active"}}`)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	out, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "nightly", "--interval", "weeklyOn", "--day", "0", "--at", "13:00", "--timezone", "America/Toronto", "--retention", "2", "--immediate", "--api-url", s.URL, "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	if got["service"] != "Virtual Machine" || got["action"] != "VM Backup" || got["take_one_immediately"].(float64) != 1 {
		t.Fatalf("request=%v", got)
	}
	if got["day"].(float64) != 0 {
		t.Fatalf("day=%v", got["day"])
	}
	if !strings.Contains(out, `"timezone": "America/Toronto"`) {
		t.Fatalf("output=%s", out)
	}
}

func TestVMBackupScheduleValidation(t *testing.T) {
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "x", "--interval", "monthlyOn", "--day", "0", "--at", "13:00", "--timezone", "UTC", "--retention", "1")
	if err == nil || !strings.Contains(err.Error(), "--day must be 1 through 28") {
		t.Fatalf("err=%v", err)
	}
}

func TestVMBackupScheduleHourlyDefaultsTime(t *testing.T) {
	var got map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":[{"slug":"vm-1"}]}`)
			return
		}
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(w, `{"data":{"id":"policy-1"}}`)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "hourly", "--interval", "hourly", "--timezone", "UTC", "--retention", "1", "--api-url", s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got["at"] != "00:00" {
		t.Fatalf("at=%v", got["at"])
	}
}

func TestVMBackupScheduleCreateAllowsUnnamedPolicy(t *testing.T) {
	var got map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":[{"slug":"vm-1"}]}`)
			return
		}
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&got)
			fmt.Fprint(w, `{"data":{"id":"p","name":"Scheduler: VirtualMachineBackup for VirtualMachine(vm-1)"}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--interval", "dailyAt", "--at", "13:00", "--timezone", "UTC", "--retention", "1", "--api-url", s.URL)
	if err != nil || got["name"] != nil || got["description"] != nil {
		t.Fatalf("err=%v request=%v", err, got)
	}
}

func TestVMBackupScheduleRejectsInvalidTimezone(t *testing.T) {
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "x", "--interval", "dailyAt", "--at", "13:00", "--timezone", "not/a-zone", "--retention", "1")
	if err == nil || !strings.Contains(err.Error(), "valid IANA timezone") {
		t.Fatalf("err=%v", err)
	}
}

func TestVMBackupScheduleValidationBoundaries(t *testing.T) {
	cases := []struct {
		interval, at, tz string
		day, retention   int
	}{
		{"dailyAt", "13:17", "UTC", -1, 1}, {"dailyAt", "13:00", "Local", -1, 1}, {"weeklyOn", "13:00", "UTC", 7, 1}, {"monthlyOn", "13:00", "UTC", 29, 1}, {"dailyAt", "13:00", "UTC", -1, 0},
	}
	for _, tc := range cases {
		if _, err := validateSchedule(tc.interval, tc.at, tc.tz, tc.day, tc.retention); err == nil {
			t.Fatalf("expected validation error for %+v", tc)
		}
	}
	if _, err := validateSchedule("monthlyOn", "13:00", "UTC", 28, 1); err != nil {
		t.Fatal(err)
	}
}

func TestVMBackupSchedulePauseRejectsScopeAndMissingReferencesBeforeWrite(t *testing.T) {
	for _, body := range []string{
		`{"data":{"id":"p","action":"Virtual Machine Backup","region":{"slug":"other"},"project":{"slug":"test"}}}`,
		`{"data":{"id":"p","action":"Virtual Machine Backup"}}`,
	} {
		writes := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				fmt.Fprint(w, body)
				return
			}
			writes++
			http.NotFound(w, r)
		}))
		t.Setenv("ZCP_BEARER_TOKEN", "test")
		t.Setenv("ZCP_REGION", "yul-1")
		t.Setenv("ZCP_PROJECT", "test")
		_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "pause", "p", "--api-url", s.URL)
		s.Close()
		if err == nil || writes != 0 {
			t.Fatalf("body=%s err=%v writes=%d", body, err, writes)
		}
	}
}

func TestVMBackupScheduleDeleteJSONReturnsActionResponse(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"data":{"action":"Virtual Machine Backup","region":{"slug":"yul-1"},"project":{"slug":"test"}}}`)
			return
		}
		if r.Method == http.MethodDelete {
			fmt.Fprint(w, `{"status":"success","message":"deleted"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	t.Setenv("ZCP_REGION", "yul-1")
	t.Setenv("ZCP_PROJECT", "test")
	out, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "delete", "p", "--yes", "--api-url", s.URL, "--output", "json")
	if err != nil || !strings.Contains(out, `"message": "deleted"`) {
		t.Fatalf("err=%v out=%s", err, out)
	}
}

func TestVMBackupScheduleOutputFormatHonorsUppercaseEnvironment(t *testing.T) {
	t.Setenv("ZCP_OUTPUT", "TABLE")
	cmd := newTestRoot()
	if got := outputFormat(cmd); string(got) != "table" {
		t.Fatalf("format=%q", got)
	}
}

func TestVMBackupScheduleCreateRejectsVMOutsideScopeBeforePost(t *testing.T) {
	posts := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/virtual-machines" {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":[{"slug":"other"}]}`)
			return
		}
		if r.Method == http.MethodPost {
			posts++
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	t.Setenv("ZCP_REGION", "yul-1")
	t.Setenv("ZCP_PROJECT", "test")
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "x", "--interval", "dailyAt", "--at", "13:00", "--timezone", "UTC", "--retention", "1", "--api-url", s.URL)
	if err == nil || posts != 0 {
		t.Fatalf("err=%v posts=%d", err, posts)
	}
}

func TestVMBackupScheduleRunNowUsesAcknowledgement(t *testing.T) {
	writes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"data":{"id":"p","action":"Virtual Machine Backup","region":{"slug":"yul-1"},"project":{"slug":"test"}}}`)
			return
		}
		if r.Method == http.MethodPost {
			writes++
			fmt.Fprint(w, `{"status":"success","message":"Virtual Machine Backup triggered successfully for vm"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	t.Setenv("ZCP_REGION", "yul-1")
	t.Setenv("ZCP_PROJECT", "test")
	out, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "run-now", "p", "--yes", "--api-url", s.URL, "--output", "json")
	if err != nil || writes != 1 || !strings.Contains(out, `"message": "Virtual Machine Backup triggered successfully for vm"`) {
		t.Fatalf("err=%v writes=%d out=%s", err, writes, out)
	}
}

func TestVMBackupScheduleCreateDoesNotRetryAfterServerFailure(t *testing.T) {
	posts := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":[{"slug":"vm-1"}]}`)
			return
		}
		if r.Method == http.MethodPost {
			posts++
			http.Error(w, "failed", http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "create", "vm-1", "--name", "x", "--interval", "dailyAt", "--at", "13:00", "--timezone", "UTC", "--retention", "1", "--api-url", s.URL)
	if err == nil || posts != 1 || !strings.Contains(err.Error(), "schedule list") {
		t.Fatalf("err=%v posts=%d", err, posts)
	}
}

func TestVMBackupSchedulePauseRejectsWrongActionBeforeWrite(t *testing.T) {
	writes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"data":{"id":"p","action":"Block Storage Snapshot","region":{"slug":"yul-1"},"project":{"slug":"test"}}}`)
			return
		}
		writes++
		http.NotFound(w, r)
	}))
	defer s.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test")
	t.Setenv("ZCP_REGION", "yul-1")
	t.Setenv("ZCP_PROJECT", "test")
	_, _, err := execCmd(t, NewVMBackupCmd(), "schedule", "pause", "p", "--api-url", s.URL)
	if err == nil || writes != 0 {
		t.Fatalf("err=%v writes=%d", err, writes)
	}
}
