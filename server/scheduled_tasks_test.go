package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestHandleScheduledTasksHTTP drives the real GET handler end-to-end and
// asserts the platform-support flag matches systemctl availability. On macOS
// (no systemctl) this confirms graceful degradation: 200 + empty list.
func TestHandleScheduledTasksHTTP(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks", nil)
	rec := httptest.NewRecorder()
	s.handleScheduledTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp ScheduledTasksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}
	_, hasSystemctl := lookSystemctl()
	if resp.PlatformSupported != hasSystemctl {
		t.Errorf("platformSupported = %v, want %v", resp.PlatformSupported, hasSystemctl)
	}
	if resp.Tasks == nil {
		t.Errorf("tasks should be a non-nil slice, got nil")
	}
	if !hasSystemctl && len(resp.Tasks) != 0 {
		t.Errorf("expected no tasks without systemctl, got %d", len(resp.Tasks))
	}
}

func lookSystemctl() (string, bool) {
	p, err := exec.LookPath("systemctl")
	return p, err == nil
}

func TestHandleScheduledTasksRejectsNonGET(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/scheduled-tasks", nil)
	rec := httptest.NewRecorder()
	s.handleScheduledTasks(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandleScheduledTaskRuns(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	runsDir := filepath.Join(dir, "shelley", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"ts":"2026-07-14T09:00:00-05:00","conversationId":"c1","slug":"s1"}
not-json-skip-me
{"ts":"2026-07-15T09:00:00-05:00","conversationId":"c2","slug":"s2"}
`
	if err := os.WriteFile(filepath.Join(runsDir, "shelley-daily.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks/shelley-daily/runs", nil)
	req.SetPathValue("name", "shelley-daily")
	rec := httptest.NewRecorder()
	s.handleScheduledTaskRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var runs []ScheduledRunAPI
	if err := json.Unmarshal(rec.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("got %d runs, want 2 (malformed line skipped)", len(runs))
	}
	// Newest first.
	if runs[0].ConversationID != "c2" || runs[1].ConversationID != "c1" {
		t.Errorf("runs not newest-first: %+v", runs)
	}
}

func TestHandleScheduledTaskRunsMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks/shelley-none/runs", nil)
	req.SetPathValue("name", "shelley-none")
	rec := httptest.NewRecorder()
	s.handleScheduledTaskRuns(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Errorf("body = %q, want empty array", got)
	}
}

func TestHandleScheduledTaskRunsInvalidName(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks/evil/runs", nil)
	req.SetPathValue("name", "../evil")
	rec := httptest.NewRecorder()
	s.handleScheduledTaskRuns(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleDeleteScheduledTaskInvalidName(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodDelete, "/api/scheduled-tasks/evil", nil)
	req.SetPathValue("name", "../evil")
	rec := httptest.NewRecorder()
	s.handleDeleteScheduledTask(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid name", rec.Code)
	}
}

func TestParseUnitValue(t *testing.T) {
	timer := `[Unit]
Description=Shelley daily standup

[Timer]
OnCalendar=*-*-* 09:00:00
Persistent=true

[Install]
WantedBy=timers.target
`
	if got := parseUnitValue(timer, "OnCalendar"); got != "*-*-* 09:00:00" {
		t.Errorf("OnCalendar = %q, want %q", got, "*-*-* 09:00:00")
	}
	if got := parseUnitValue(timer, "Description"); got != "Shelley daily standup" {
		t.Errorf("Description = %q, want %q", got, "Shelley daily standup")
	}
	if got := parseUnitValue(timer, "Missing"); got != "" {
		t.Errorf("Missing = %q, want empty", got)
	}
}

func TestParseUnitValueIgnoresComments(t *testing.T) {
	body := `# OnCalendar=commented-out
OnCalendar=weekly
`
	if got := parseUnitValue(body, "OnCalendar"); got != "weekly" {
		t.Errorf("OnCalendar = %q, want %q", got, "weekly")
	}
}

func TestParseExecStart(t *testing.T) {
	tests := []struct {
		name       string
		execStart  string
		wantPrompt string
		wantCwd    string
	}{
		{
			name:       "prompt and cwd",
			execStart:  `/usr/bin/shelley client chat -p 'Check the CI status' -cwd '/home/me/repos/app'`,
			wantPrompt: "Check the CI status",
			wantCwd:    "/home/me/repos/app",
		},
		{
			name:       "prompt with escaped quote",
			execStart:  `shelley client chat -p 'It\'s time' -cwd '/tmp'`,
			wantPrompt: `It\'s time`,
			wantCwd:    "/tmp",
		},
		{
			name:       "prompt only",
			execStart:  `shelley client chat -p 'do the thing'`,
			wantPrompt: "do the thing",
			wantCwd:    "",
		},
		{
			name:       "no match",
			execStart:  `/bin/true`,
			wantPrompt: "",
			wantCwd:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, cwd := parseExecStart(tt.execStart)
			if prompt != tt.wantPrompt {
				t.Errorf("prompt = %q, want %q", prompt, tt.wantPrompt)
			}
			if cwd != tt.wantCwd {
				t.Errorf("cwd = %q, want %q", cwd, tt.wantCwd)
			}
		})
	}
}

func TestHumanizeSchedule(t *testing.T) {
	tests := map[string]string{
		"*-*-* 18:00:00 America/Chicago": "Daily 18:00:00 America/Chicago",
		"*-*-* 09:00:00":                 "Daily 09:00:00",
		"Mon *-*-* 08:00:00":             "Weekly on Monday 08:00:00",
		"Mon,Wed,Fri *-*-* 08:00:00":     "Mon, Wed, Fri 08:00:00",
		"Mon..Fri *-*-* 08:00:00":        "Mon–Fri 08:00:00",
		"*-*-01 00:00:00":                "Monthly on the 1st 00:00:00",
		"*-*-23 12:30:00":                "Monthly on the 23rd 12:30:00",
		"daily":                          "Daily",
		"hourly":                         "Hourly",
		"":                               "",
		// Unrecognized -> returned unchanged.
		"*-05-01 00:00:00": "*-05-01 00:00:00",
	}
	for in, want := range tests {
		if got := humanizeSchedule(in); got != want {
			t.Errorf("humanizeSchedule(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTimerStatus(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour).UnixMicro()
	past := now.Add(-time.Hour).UnixMicro()

	tests := []struct {
		name        string
		activeState string
		nextUsec    int64
		want        string
	}{
		{"armed recurring", "active", future, "active"},
		{"one-shot already fired (elapsed)", "active", 0, "completed"},
		{"active but next in the past", "active", past, "completed"},
		{"disabled/not loaded", "inactive", 0, "inactive"},
		{"failed unit", "failed", future, "inactive"},
	}
	for _, tt := range tests {
		if got := timerStatus(tt.activeState, tt.nextUsec, now); got != tt.want {
			t.Errorf("%s: timerStatus(%q, %d) = %q, want %q", tt.name, tt.activeState, tt.nextUsec, got, tt.want)
		}
	}
}

func TestParseShowProps(t *testing.T) {
	out := "ActiveState=active\nNextElapseUSecRealtime=1752620400000000\nLastTriggerUSec=0\n"
	props := parseShowProps(out)
	if props["ActiveState"] != "active" {
		t.Errorf("ActiveState = %q, want active", props["ActiveState"])
	}
	if props["NextElapseUSecRealtime"] != "1752620400000000" {
		t.Errorf("NextElapseUSecRealtime = %q", props["NextElapseUSecRealtime"])
	}
}

func TestParseUSec(t *testing.T) {
	tests := map[string]int64{
		"1752620400000000":     1752620400000000,
		"0":                    0,
		"":                     0,
		"18446744073709551615": 0, // uint64 max sentinel overflows int64 -> 0
		"not-a-number":         0,
	}
	for in, want := range tests {
		if got := parseUSec(in); got != want {
			t.Errorf("parseUSec(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestScheduledUnitNameRE(t *testing.T) {
	valid := []string{"shelley-daily", "shelley-weekly_report", "shelley-a-b-c"}
	invalid := []string{"", "daily", "shelley-", "shelley-../etc", "shelley-a b", "shelley-a.timer"}
	for _, v := range valid {
		if !scheduledUnitNameRE.MatchString(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}
	for _, v := range invalid {
		if scheduledUnitNameRE.MatchString(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}
