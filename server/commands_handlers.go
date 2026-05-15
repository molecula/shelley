package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"shelley.exe.dev/commands"
	"shelley.exe.dev/gitstate"
	"shelley.exe.dev/skills"
)

// BuiltinCommand describes a UI-level slash command (executed entirely by
// the frontend).
type BuiltinCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// builtinCommands is the canonical list of UI-action slash commands.
var builtinCommands = []BuiltinCommand{
	{Name: "new", Description: "Start a new conversation", Action: "new-conversation"},
	{Name: "clear", Description: "Archive this conversation and start a new one", Action: "clear"},
	{Name: "model", Description: "Open the model picker", Action: "open-model-picker"},
	{Name: "help", Description: "Show available slash commands", Action: "help"},
}

// PaletteSkill is the subset of skill metadata sent to the palette.
type PaletteSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsBuiltin   bool   `json:"is_builtin"`
}

// CommandsResponse is the payload for GET /api/commands.
type CommandsResponse struct {
	Builtins     []BuiltinCommand   `json:"builtins"`
	UserCommands []commands.Command `json:"user_commands"`
	Skills       []PaletteSkill     `json:"skills"`
}

// handleCommands returns slash command suggestions for the palette.
// Query params:
//   cwd: working directory (optional, defaults to server cwd)
func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	gitRoot := ""
	if gs := gitstate.GetGitState(cwd); gs != nil {
		gitRoot = gs.Worktree
	}

	userCmds := commands.ListAll(cwd, gitRoot)
	if userCmds == nil {
		userCmds = []commands.Command{}
	}
	sort.Slice(userCmds, func(i, j int) bool { return userCmds[i].Name < userCmds[j].Name })

	var paletteSkills []PaletteSkill
	for _, sk := range skills.ListAll(cwd, gitRoot) {
		paletteSkills = append(paletteSkills, PaletteSkill{
			Name:        sk.Name,
			Description: sk.Description,
			IsBuiltin:   sk.Path == "",
		})
	}
	if paletteSkills == nil {
		paletteSkills = []PaletteSkill{}
	}

	resp := CommandsResponse{
		Builtins:     builtinCommands,
		UserCommands: userCmds,
		Skills:       paletteSkills,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UserSkill is the response shape for the skills viewer.
type UserSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Scope       string `json:"scope"` // "project", "user", or "builtin"
}

// handleUserSkills returns user-defined (non-builtin) skills for the viewer.
func (s *Server) handleUserSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	gitRoot := ""
	if gs := gitstate.GetGitState(cwd); gs != nil {
		gitRoot = gs.Worktree
	}

	var projectDirs []string
	projectDirs = append(projectDirs, skills.ProjectSkillsDirs(cwd, gitRoot)...)
	projectSet := make(map[string]bool)
	for _, d := range projectDirs {
		projectSet[d] = true
	}

	var out []UserSkill
	for _, sk := range skills.ListAll(cwd, gitRoot) {
		scope := "user"
		if sk.Path == "" {
			scope = "builtin"
		} else {
			parent := filepath.Dir(sk.Path)
			if projectSet[filepath.Dir(parent)] || projectSet[parent] {
				scope = "project"
			}
		}
		out = append(out, UserSkill{
			Name:        sk.Name,
			Description: sk.Description,
			Path:        sk.Path,
			Scope:       scope,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if out == nil {
		out = []UserSkill{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleUserSkillContent returns the full SKILL.md content for a named skill.
func (s *Server) handleUserSkillContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}

	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	gitRoot := ""
	if gs := gitstate.GetGitState(cwd); gs != nil {
		gitRoot = gs.Worktree
	}

	for _, sk := range skills.ListAll(cwd, gitRoot) {
		if sk.Name != name {
			continue
		}
		var content string
		if sk.Path == "" {
			// Built-in skill: reconstruct file from frontmatter + embedded body.
			content = "---\nname: " + sk.Name + "\ndescription: " + sk.Description + "\n---\n\n" + sk.Body
		} else {
			data, err := os.ReadFile(sk.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			content = string(data)
		}
		resp := struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Path        string `json:"path"`
			Content     string `json:"content"`
		}{
			Name:        sk.Name,
			Description: sk.Description,
			Path:        sk.Path,
			Content:     strings.TrimRight(content, "\n") + "\n",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	http.Error(w, "skill not found", http.StatusNotFound)
}
