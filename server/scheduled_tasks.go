package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type ScheduledTask struct {
	Name        string `json:"name"`         // unit base name without .timer/.service (e.g. "shelley-foo")
	Description string `json:"description"`  // service Description=
	ExecStart   string `json:"exec_start"`   // service ExecStart=
	NextFire    string `json:"next_fire"`    // timer NextElapseUSecRealtime as ISO time, or empty
	LastFire    string `json:"last_fire"`    // timer LastTriggerUSec as ISO time, or empty
	TimerState  string `json:"timer_state"`  // active/inactive/etc
	OnCalendar  string `json:"on_calendar"`  // OnCalendar=
	Persistent  bool   `json:"persistent"`   // Persistent=
}

func (s *Server) handleScheduledTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listScheduledTasks(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScheduledTask(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/scheduled-tasks/")
	if name == "" || strings.ContainsAny(name, "/ \t") {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(name, "shelley-") {
		http.Error(w, "only shelley-* units may be managed", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := deleteShelleyUnit(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listScheduledTasks(w http.ResponseWriter, r *http.Request) {
	// list-timers shows both static and runtime-created timers; list-unit-files misses some.
	out, err := exec.Command("systemctl", "--user", "list-timers", "--all", "--no-legend", "--no-pager").Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("list timers: %v", err), http.StatusInternalServerError)
		return
	}
	var tasks []ScheduledTask
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var timerUnit string
		for _, f := range strings.Fields(line) {
			if strings.HasSuffix(f, ".timer") {
				timerUnit = f
				break
			}
		}
		if timerUnit == "" || seen[timerUnit] {
			continue
		}
		seen[timerUnit] = true
		base := strings.TrimSuffix(timerUnit, ".timer")
		t := ScheduledTask{Name: base}
		fillFromShow(&t, base+".timer", []string{"NextElapseUSecRealtime", "LastTriggerUSec", "ActiveState", "OnCalendar", "Persistent"})
		fillFromShow(&t, base+".service", []string{"Description", "ExecStart"})
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []ScheduledTask{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func fillFromShow(t *ScheduledTask, unit string, props []string) {
	args := []string{"--user", "show", unit, "--property=" + strings.Join(props, ",")}
	out, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "NextElapseUSecRealtime":
			t.NextFire = v
		case "LastTriggerUSec":
			t.LastFire = v
		case "ActiveState":
			t.TimerState = v
		case "OnCalendar":
			// format: "{ OnCalendar=... ; next_elapse=... }" — keep raw
			t.OnCalendar = v
		case "Persistent":
			t.Persistent = v == "yes"
		case "Description":
			t.Description = v
		case "ExecStart":
			// systemd ExecStart show output is structured; extract argv portion
			t.ExecStart = extractExecStart(v)
		}
	}
}

// extractExecStart pulls the argv= portion from systemd's ExecStart show output.
func extractExecStart(v string) string {
	// Example: { path=/bin/sh ; argv[]=/bin/sh -c "echo hi" ; ignore_errors=no ; ... }
	const marker = "argv[]="
	i := strings.Index(v, marker)
	if i < 0 {
		return v
	}
	rest := v[i+len(marker):]
	if j := strings.Index(rest, " ; "); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func deleteShelleyUnit(base string) error {
	timer := base + ".timer"
	service := base + ".service"
	// Best-effort stop+disable; ignore errors from already-stopped units.
	_ = exec.Command("systemctl", "--user", "stop", timer).Run()
	_ = exec.Command("systemctl", "--user", "stop", service).Run()
	_ = exec.Command("systemctl", "--user", "disable", timer).Run()

	// Locate and remove unit files under $XDG_CONFIG_HOME/systemd/user.
	out, err := exec.Command("systemctl", "--user", "show", "-p", "FragmentPath", timer).Output()
	if err == nil {
		path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "FragmentPath="))
		if path != "" {
			_ = exec.Command("rm", "-f", path).Run()
		}
	}
	out, err = exec.Command("systemctl", "--user", "show", "-p", "FragmentPath", service).Output()
	if err == nil {
		path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "FragmentPath="))
		if path != "" {
			_ = exec.Command("rm", "-f", path).Run()
		}
	}
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	_ = exec.Command("systemctl", "--user", "reset-failed").Run()
	return nil
}
