package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ScheduledTaskAPI is the response shape for the scheduled-tasks viewer. Each
// task corresponds to a `shelley-<name>.timer` + `shelley-<name>.service` pair
// of systemd user units, as created by the builtin `/schedule` skill.
type ScheduledTaskAPI struct {
	// Name is the unit base name including the "shelley-" prefix, e.g.
	// "shelley-daily-standup". It is the identifier used for deletion.
	Name string `json:"name"`
	// Schedule is the raw systemd OnCalendar expression from the .timer unit.
	Schedule string `json:"schedule"`
	// ScheduleLabel is a human-legible rendering of Schedule (e.g. "Daily
	// 18:00:00 America/Chicago"), falling back to the raw expression when it
	// can't be interpreted.
	ScheduleLabel string `json:"scheduleLabel"`
	// Prompt and Cwd are extracted from the .service unit's ExecStart line
	// (`shelley client chat -p '<prompt>' -cwd '<cwd>'`).
	Prompt string `json:"prompt"`
	Cwd    string `json:"cwd"`
	// NextRun / LastRun are human-readable timestamps from the timer's live
	// state; NextRun is empty when there is no upcoming run.
	NextRun string `json:"nextRun"`
	LastRun string `json:"lastRun"`
	// Status is one of "active" (armed with an upcoming run), "completed" (a
	// one-shot timer that has already fired, nothing upcoming), or "inactive"
	// (not loaded/enabled).
	Status string `json:"status"`
}

// ScheduledTasksResponse wraps the task list with a platform-support flag.
// systemd user timers only exist on Linux; on other platforms (or where
// `systemctl` is unavailable) PlatformSupported is false and Tasks is empty.
type ScheduledTasksResponse struct {
	PlatformSupported bool               `json:"platformSupported"`
	Tasks             []ScheduledTaskAPI `json:"tasks"`
}

// scheduledUnitNameRE guards the {name} path segment so a DELETE can only
// target shelley-prefixed units and never escape the unit directory.
var scheduledUnitNameRE = regexp.MustCompile(`^shelley-[A-Za-z0-9_-]+$`)

// systemdUserUnitDir returns the per-user systemd unit directory
// (~/.config/systemd/user), honoring XDG_CONFIG_HOME.
func systemdUserUnitDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "systemd", "user"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// handleScheduledTasks handles GET /api/scheduled-tasks.
func (s *Server) handleScheduledTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := ScheduledTasksResponse{Tasks: []ScheduledTaskAPI{}}

	// If systemctl is unavailable (e.g. macOS), scheduling isn't supported.
	if _, err := exec.LookPath("systemctl"); err != nil {
		writeJSON(w, resp)
		return
	}
	resp.PlatformSupported = true

	unitDir, err := systemdUserUnitDir()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to locate unit directory: %v", err), http.StatusInternalServerError)
		return
	}

	timers, err := filepath.Glob(filepath.Join(unitDir, "shelley-*.timer"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list timers: %v", err), http.StatusInternalServerError)
		return
	}

	unitNames := make([]string, 0, len(timers))
	for _, timerPath := range timers {
		unitNames = append(unitNames, filepath.Base(timerPath))
	}
	runtime := timerRuntimeState(r.Context(), unitNames)

	tasks := make([]ScheduledTaskAPI, 0, len(timers))
	for _, timerPath := range timers {
		base := strings.TrimSuffix(filepath.Base(timerPath), ".timer")
		task := ScheduledTaskAPI{Name: base, Status: "inactive"}

		if data, err := os.ReadFile(timerPath); err == nil {
			task.Schedule = parseUnitValue(string(data), "OnCalendar")
			task.ScheduleLabel = humanizeSchedule(task.Schedule)
		}
		servicePath := filepath.Join(unitDir, base+".service")
		if data, err := os.ReadFile(servicePath); err == nil {
			execStart := parseUnitValue(string(data), "ExecStart")
			task.Prompt, task.Cwd = parseExecStart(execStart)
		}
		if rt, ok := runtime[base+".timer"]; ok {
			task.NextRun = rt.next
			task.LastRun = rt.last
			task.Status = rt.status
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })
	resp.Tasks = tasks

	writeJSON(w, resp)
}

// handleDeleteScheduledTask handles DELETE /api/scheduled-tasks/{name}.
func (s *Server) handleDeleteScheduledTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.PathValue("name")
	if !scheduledUnitNameRE.MatchString(name) {
		http.Error(w, "invalid task name", http.StatusBadRequest)
		return
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		http.Error(w, "scheduling is not supported on this platform", http.StatusServiceUnavailable)
		return
	}

	unitDir, err := systemdUserUnitDir()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to locate unit directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Stop + disable the timer (ignore errors: the unit may already be gone).
	ctx := r.Context()
	if out, err := runSystemctl(ctx, "--user", "disable", "--now", name+".timer"); err != nil {
		s.logger.Warn("scheduled task: disable timer failed", "name", name, "err", err, "out", out)
	}

	// Remove the unit files.
	var removeErr error
	for _, suffix := range []string{".timer", ".service"} {
		path := filepath.Join(unitDir, name+suffix)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			removeErr = err
		}
	}
	if removeErr != nil {
		http.Error(w, fmt.Sprintf("Failed to remove unit files: %v", removeErr), http.StatusInternalServerError)
		return
	}

	if out, err := runSystemctl(ctx, "--user", "daemon-reload"); err != nil {
		s.logger.Warn("scheduled task: daemon-reload failed", "err", err, "out", out)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ScheduledRunAPI is one recorded run of a scheduled task. Each run is a
// separate conversation the task spawned; the UI links to it.
type ScheduledRunAPI struct {
	Ts             string `json:"ts"`
	ConversationID string `json:"conversationId"`
	Slug           string `json:"slug"`
}

// scheduledRunsDir returns ~/.config/shelley/runs, honoring XDG_CONFIG_HOME.
// This must match the path the client's -schedule-name writer uses.
func scheduledRunsDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "shelley", "runs"), nil
}

// handleScheduledTaskRuns handles GET /api/scheduled-tasks/{name}/runs. It
// reads the per-task JSONL log the client appends on each firing, newest first.
func (s *Server) handleScheduledTaskRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	if !scheduledUnitNameRE.MatchString(name) {
		http.Error(w, "invalid task name", http.StatusBadRequest)
		return
	}

	runs := []ScheduledRunAPI{}
	dir, err := scheduledRunsDir()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to locate runs directory: %v", err), http.StatusInternalServerError)
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".jsonl"))
	if err != nil {
		// No log yet (task never ran, or predates run recording): empty list.
		writeJSON(w, runs)
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var run ScheduledRunAPI
		if err := json.Unmarshal([]byte(line), &run); err != nil {
			continue // skip malformed lines
		}
		runs = append(runs, run)
	}
	// Newest first.
	for i, j := 0, len(runs)-1; i < j; i, j = i+1, j-1 {
		runs[i], runs[j] = runs[j], runs[i]
	}
	writeJSON(w, runs)
}

// parseUnitValue returns the value of the last occurrence of `key=` in a
// systemd unit file body. systemd uses the last assignment within a section,
// which is a good-enough heuristic for the single-value keys we read.
func parseUnitValue(body, key string) string {
	var value string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if eq := strings.IndexByte(line, '='); eq >= 0 {
			if strings.TrimSpace(line[:eq]) == key {
				value = strings.TrimSpace(line[eq+1:])
			}
		}
	}
	return value
}

// execStartPromptRE / execStartCwdRE pull the single-quoted values out of an
// ExecStart line like:
//
//	shelley client chat -p '<prompt>' -cwd '<dir>'
//
// The skill always single-quotes these arguments.
var (
	execStartPromptRE = regexp.MustCompile(`-p\s+'((?:[^'\\]|\\.)*)'`)
	execStartCwdRE    = regexp.MustCompile(`-cwd\s+'((?:[^'\\]|\\.)*)'`)
)

// parseExecStart extracts the prompt and cwd from a service ExecStart value.
func parseExecStart(execStart string) (prompt, cwd string) {
	if m := execStartPromptRE.FindStringSubmatch(execStart); m != nil {
		prompt = m[1]
	}
	if m := execStartCwdRE.FindStringSubmatch(execStart); m != nil {
		cwd = m[1]
	}
	return prompt, cwd
}

var weekdayFull = map[string]string{
	"mon": "Monday", "tue": "Tuesday", "wed": "Wednesday", "thu": "Thursday",
	"fri": "Friday", "sat": "Saturday", "sun": "Sunday",
}

// humanizeSchedule renders a systemd OnCalendar expression as a friendly label,
// e.g. "*-*-* 18:00:00 America/Chicago" -> "Daily 18:00:00 America/Chicago".
// Returns the raw expression unchanged for anything it doesn't recognize.
func humanizeSchedule(expr string) string {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return ""
	}
	// systemd calendar shortcuts.
	switch strings.ToLower(raw) {
	case "minutely":
		return "Every minute"
	case "hourly":
		return "Hourly"
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly (Mon 00:00)"
	case "monthly":
		return "Monthly (1st, 00:00)"
	case "quarterly":
		return "Quarterly"
	case "semiannually":
		return "Twice a year"
	case "yearly", "annually":
		return "Yearly"
	}

	fields := strings.Fields(raw)
	timeIdx := -1
	for i, f := range fields {
		if strings.Contains(f, ":") {
			timeIdx = i
			break
		}
	}
	if timeIdx == -1 {
		return raw // no time component we understand
	}
	timeStr := fields[timeIdx]
	tz := ""
	if timeIdx+1 < len(fields) {
		tz = strings.Join(fields[timeIdx+1:], " ")
	}

	var dateTok, dowTok string
	for i := 0; i < timeIdx; i++ {
		if strings.Contains(fields[i], "-") {
			dateTok = fields[i]
		} else {
			dowTok = fields[i]
		}
	}

	var freq string
	switch {
	case dowTok != "":
		freq = humanizeWeekdays(dowTok)
	case dateTok == "" || dateTok == "*-*-*":
		freq = "Daily"
	default:
		parts := strings.Split(dateTok, "-")
		if len(parts) == 3 && parts[0] == "*" && parts[1] == "*" && parts[2] != "*" {
			freq = "Monthly on the " + ordinal(parts[2])
		}
	}
	if freq == "" {
		return raw
	}

	out := freq + " " + timeStr
	if tz != "" {
		out += " " + tz
	}
	return out
}

// humanizeWeekdays turns a systemd weekday token (e.g. "Mon", "Mon..Fri",
// "Mon,Wed,Fri") into a readable phrase. Returns "" if it can't.
func humanizeWeekdays(tok string) string {
	days := strings.Split(strings.ToLower(tok), ",")
	if len(days) == 1 && !strings.Contains(days[0], "..") {
		if full, ok := weekdayFull[days[0]]; ok {
			return "Weekly on " + full
		}
		return ""
	}
	titleCase := func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}
	var parts []string
	for _, d := range days {
		if strings.Contains(d, "..") {
			r := strings.SplitN(d, "..", 2)
			parts = append(parts, titleCase(r[0])+"–"+titleCase(r[1]))
		} else {
			parts = append(parts, titleCase(d))
		}
	}
	return strings.Join(parts, ", ")
}

// ordinal renders a day-of-month string (possibly zero-padded) as "1st", "2nd",
// "23rd", etc. Falls back to the trimmed input if it isn't a number.
func ordinal(day string) string {
	n, err := strconv.Atoi(strings.TrimSpace(day))
	if err != nil {
		return day
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

type timerRuntime struct {
	next   string
	last   string
	status string
}

// timerStatus derives a task's status from the timer unit's live ActiveState
// and its next-elapse time. A one-shot timer that has already fired stays
// ActiveState=active (systemd keeps it "active (elapsed)") but has no upcoming
// run, which we surface as "completed" rather than the misleading "active".
func timerStatus(activeState string, nextUsec int64, now time.Time) string {
	if activeState != "active" {
		return "inactive"
	}
	if nextUsec > 0 && time.UnixMicro(nextUsec).After(now) {
		return "active"
	}
	return "completed"
}

// timerRuntimeState queries each timer unit's live state via `systemctl --user
// show`, keyed by timer unit name (e.g. "shelley-foo.timer"). `show` is used
// instead of `list-timers --output=json` because that command's next/last
// fields are integer microseconds (or null), not the strings an earlier
// version assumed — a mismatch that silently blanked all runtime state.
// Best-effort: units that error out are simply omitted.
func timerRuntimeState(ctx context.Context, units []string) map[string]timerRuntime {
	result := map[string]timerRuntime{}
	for _, unit := range units {
		out, err := runSystemctl(ctx, "--user", "show", unit,
			"--property=ActiveState",
			"--property=NextElapseUSecRealtime",
			"--property=LastTriggerUSec")
		if err != nil {
			continue
		}
		props := parseShowProps(out)
		now := time.Now()
		next := parseUSec(props["NextElapseUSecRealtime"])
		rt := timerRuntime{status: timerStatus(props["ActiveState"], next, now)}
		if next > 0 && time.UnixMicro(next).After(now) {
			rt.next = time.UnixMicro(next).Format("Mon 2006-01-02 15:04:05 MST")
		}
		if usec := parseUSec(props["LastTriggerUSec"]); usec > 0 {
			rt.last = time.UnixMicro(usec).Format("Mon 2006-01-02 15:04:05 MST")
		}
		result[unit] = rt
	}
	return result
}

// parseShowProps parses the `key=value` lines emitted by `systemctl show`.
func parseShowProps(out string) map[string]string {
	props := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if eq := strings.IndexByte(line, '='); eq >= 0 {
			props[line[:eq]] = strings.TrimSpace(line[eq+1:])
		}
	}
	return props
}

// parseUSec parses a microseconds-since-epoch value from a `systemctl show`
// property. Returns 0 for empty, unparseable, or systemd's "unset" sentinels
// (0 or the uint64 max, which overflows int64 and thus fails to parse).
func parseUSec(v string) int64 {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// runSystemctl runs systemctl with the given args and returns combined output.
func runSystemctl(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// writeJSON encodes v as JSON with the appropriate content type.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
