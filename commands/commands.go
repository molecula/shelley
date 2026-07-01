// Package commands discovers user-defined slash commands.
//
// Commands are markdown files placed in `~/.claude/commands/<name>.md`
// (global) or `<repo>/.claude/commands/<name>.md` (project). The file's YAML
// frontmatter provides metadata and the body is the prompt template.
//
// Frontmatter fields:
//   description: short one-line description shown in the palette.
//   argument-hint: (optional) hint shown next to the command name, e.g. "<file>".
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxNameLength        = 64
	MaxDescriptionLength = 1024
)

// Command represents a parsed user-defined slash command.
type Command struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argument_hint,omitempty"`
	Body         string `json:"body"`
	Path         string `json:"path"`
	Scope        string `json:"scope"` // "project" or "user"
}

// Discover scans the given directories for `*.md` files and returns the
// commands found. Files that fail validation are skipped. Earlier directories
// take precedence on name collisions.
func Discover(dirs []string) []Command {
	var out []Command
	seen := make(map[string]bool)

	for _, dir := range dirs {
		dir = expandPath(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			cmd, err := Parse(path)
			if err != nil {
				continue
			}
			if seen[cmd.Name] {
				continue
			}
			seen[cmd.Name] = true
			out = append(out, cmd)
		}
	}
	return out
}

// Parse reads and parses a command file.
func Parse(path string) (Command, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Command{}, err
	}

	content := string(data)
	frontmatter, body := splitFrontmatter(content)

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if err := validateName(name); err != nil {
		return Command{}, err
	}

	description := frontmatter["description"]
	if len(description) > MaxDescriptionLength {
		return Command{}, fmt.Errorf("description exceeds maximum length")
	}

	return Command{
		Name:         name,
		Description:  description,
		ArgumentHint: frontmatter["argument-hint"],
		Body:         strings.TrimSpace(body),
		Path:         path,
	}, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("command name is empty")
	}
	if len(name) > MaxNameLength {
		return fmt.Errorf("command name exceeds maximum length")
	}
	for _, r := range name {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("command name contains invalid character: %q", r)
		}
	}
	return nil
}

// DefaultDirs returns the global user command directory candidates.
func DefaultDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".claude", "commands"),
		filepath.Join(home, ".config", "shelley", "commands"),
	}
	var out []string
	for _, d := range candidates {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// ProjectDirs walks up from workingDir to gitRoot (or filesystem root) and
// returns any `.claude/commands` directories found. Closest-to-working-dir
// is first so it takes precedence.
func ProjectDirs(workingDir, gitRoot string) []string {
	var dirs []string
	seen := make(map[string]bool)

	stopAt := gitRoot
	if stopAt == "" {
		stopAt = "/"
	}

	current := workingDir
	for current != "" {
		candidate := filepath.Join(current, ".claude", "commands")
		if !seen[candidate] {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				dirs = append(dirs, candidate)
				seen[candidate] = true
			}
		}
		if current == stopAt || current == "/" {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return dirs
}

// ListAll discovers commands from project and global directories.
// Project commands take precedence on name collisions.
func ListAll(workingDir, gitRoot string) []Command {
	var out []Command
	seen := make(map[string]bool)

	projectDirs := ProjectDirs(workingDir, gitRoot)
	for _, c := range Discover(projectDirs) {
		if seen[c.Name] {
			continue
		}
		c.Scope = "project"
		seen[c.Name] = true
		out = append(out, c)
	}
	for _, c := range Discover(DefaultDirs()) {
		if seen[c.Name] {
			continue
		}
		c.Scope = "user"
		seen[c.Name] = true
		out = append(out, c)
	}
	return out
}

// Render substitutes $ARGUMENTS in the command body with the given argument
// string. If the template contains no $ARGUMENTS marker and args is
// non-empty, args is appended on a new line.
func Render(body, args string) string {
	if strings.Contains(body, "$ARGUMENTS") {
		return strings.ReplaceAll(body, "$ARGUMENTS", args)
	}
	if args == "" {
		return body
	}
	return body + "\n\n" + args
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// splitFrontmatter extracts simple key: value YAML frontmatter and returns
// the body. If no frontmatter is present, returns an empty map and the
// original content as the body.
func splitFrontmatter(content string) (map[string]string, string) {
	fm := map[string]string{}
	if !strings.HasPrefix(content, "---") {
		return fm, content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return fm, content
	}
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		fm[key] = val
	}
	return fm, strings.TrimPrefix(parts[2], "\n")
}
