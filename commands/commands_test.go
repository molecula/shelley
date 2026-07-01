package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAndParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy.md")
	content := "---\ndescription: Deploy the app\nargument-hint: <env>\n---\n\nDeploy to $ARGUMENTS now.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmds := Discover([]string{dir})
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	c := cmds[0]
	if c.Name != "deploy" {
		t.Errorf("name = %q, want deploy", c.Name)
	}
	if c.Description != "Deploy the app" {
		t.Errorf("description = %q", c.Description)
	}
	if c.ArgumentHint != "<env>" {
		t.Errorf("argument hint = %q", c.ArgumentHint)
	}
	if c.Body != "Deploy to $ARGUMENTS now." {
		t.Errorf("body = %q", c.Body)
	}
}

func TestRender(t *testing.T) {
	if got := Render("Hi $ARGUMENTS", "world"); got != "Hi world" {
		t.Errorf("got %q", got)
	}
	if got := Render("Body", ""); got != "Body" {
		t.Errorf("got %q", got)
	}
	if got := Render("Body", "extra"); got != "Body\n\nextra" {
		t.Errorf("got %q", got)
	}
}

func TestListAllProjectPrecedence(t *testing.T) {
	root := t.TempDir()
	projDir := filepath.Join(root, ".claude", "commands")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	projPath := filepath.Join(projDir, "thing.md")
	if err := os.WriteFile(projPath, []byte("---\ndescription: project\n---\nproject body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmds := ListAll(root, root)
	var found *Command
	for i := range cmds {
		if cmds[i].Name == "thing" {
			found = &cmds[i]
		}
	}
	if found == nil {
		t.Fatal("expected to find 'thing'")
	}
	if found.Scope != "project" {
		t.Errorf("scope = %q", found.Scope)
	}
}
