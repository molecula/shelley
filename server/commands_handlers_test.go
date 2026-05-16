package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleCommandsIncludesUserCommand(t *testing.T) {
	t.Parallel()
	svr, _, _ := newTestServer(t)

	root := t.TempDir()
	cmdDir := filepath.Join(root, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ndescription: Run the deploy script\n---\nDeploy now.\n"
	if err := os.WriteFile(filepath.Join(cmdDir, "deploy.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/commands?cwd="+root, nil)
	w := httptest.NewRecorder()
	svr.handleCommands(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var resp CommandsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Builtins) == 0 {
		t.Error("expected built-in commands")
	}
	var found bool
	for _, c := range resp.UserCommands {
		if c.Name == "deploy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'deploy' in user_commands; got %+v", resp.UserCommands)
	}
}

func TestHandleUserSkillsIncludesAllSorted(t *testing.T) {
	t.Parallel()
	svr, _, _ := newTestServer(t)

	root := t.TempDir()
	skillDir := filepath.Join(root, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: my-skill\ndescription: A test skill\n---\n\nBody here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/user-skills?cwd="+root, nil)
	w := httptest.NewRecorder()
	svr.handleUserSkills(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var out []UserSkill
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	var found *UserSkill
	for i := range out {
		if out[i].Name == "my-skill" {
			found = &out[i]
		}
	}
	if found == nil {
		t.Fatalf("expected my-skill; got %+v", out)
	}
	// Built-ins should now appear with scope="builtin".
	var hasBuiltin bool
	for _, sk := range out {
		if sk.Scope == "builtin" {
			hasBuiltin = true
			if sk.Path != "" {
				t.Errorf("built-in skill should have empty Path: %q -> %q", sk.Name, sk.Path)
			}
		}
	}
	if !hasBuiltin {
		t.Error("expected at least one built-in skill in response")
	}
	// Verify sorted alphabetically.
	for i := 1; i < len(out); i++ {
		if out[i-1].Name > out[i].Name {
			t.Errorf("not sorted: %q > %q", out[i-1].Name, out[i].Name)
		}
	}
}

func TestHandleUserSkillContent(t *testing.T) {
	t.Parallel()
	svr, _, _ := newTestServer(t)

	root := t.TempDir()
	skillDir := filepath.Join(root, ".claude", "skills", "viewme")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: viewme\ndescription: View me\n---\n\n# Hello world\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/user-skills/viewme?cwd="+root, nil)
	req.SetPathValue("name", "viewme")
	w := httptest.NewRecorder()
	svr.handleUserSkillContent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "viewme" {
		t.Errorf("name = %q", resp.Name)
	}
	if resp.Content == "" || resp.Content[0] != '-' {
		t.Errorf("content unexpected: %q", resp.Content)
	}
}
