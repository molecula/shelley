package claudetool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shelley.exe.dev/claudetool/hashline"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func tagFor(lineNum int, text string) string {
	return hashline.FormatLineTag(lineNum, text)
}

func TestEditToolReplaceRange(t *testing.T) {
	dir := t.TempDir()
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	p := writeTestFile(t, dir, "test.go", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	lines := strings.Split(content, "\n")
	posTag := tagFor(4, lines[3])

	input := map[string]any{
		"path": "test.go",
		"edits": []map[string]any{{
			"loc": map[string]any{
				"range": map[string]string{"pos": posTag, "end": posTag},
			},
			"content": []string{`	println("goodbye")`},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("edit failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if !strings.Contains(result, "goodbye") {
		t.Fatalf("expected 'goodbye' in result, got:\n%s", result)
	}
	if strings.Contains(result, "hello") {
		t.Fatalf("expected 'hello' to be replaced, got:\n%s", result)
	}
}

func TestEditToolAppendAt(t *testing.T) {
	dir := t.TempDir()
	content := "a\nb\nc"
	p := writeTestFile(t, dir, "test.txt", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	lines := strings.Split(content, "\n")
	tag := tagFor(1, lines[0])

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{{
			"loc":     map[string]string{"append": tag},
			"content": []string{"inserted"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("edit failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "a\ninserted\nb\nc" {
		t.Fatalf("got %q", result)
	}
}

func TestEditToolPrependFile(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2"
	p := writeTestFile(t, dir, "test.txt", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{{
			"loc":     "prepend",
			"content": []string{"header"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("edit failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "header\nline1\nline2" {
		t.Fatalf("got %q", result)
	}
}

func TestEditToolHashMismatch(t *testing.T) {
	dir := t.TempDir()
	content := "hello\nworld"
	writeTestFile(t, dir, "test.txt", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{{
			"loc": map[string]any{
				"range": map[string]string{"pos": "1#ZZZ", "end": "1#ZZZ"},
			},
			"content": []string{"goodbye"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !strings.Contains(out.Error.Error(), ">>>") {
		t.Fatalf("expected mismatch error with >>>, got: %s", out.Error)
	}
}

func TestEditToolDelete(t *testing.T) {
	dir := t.TempDir()
	content := "a\nb\nc\nd"
	p := writeTestFile(t, dir, "test.txt", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	lines := strings.Split(content, "\n")
	posTag := tagFor(2, lines[1])
	endTag := tagFor(3, lines[2])

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{{
			"loc": map[string]any{
				"range": map[string]string{"pos": posTag, "end": endTag},
			},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("edit failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "a\nd" {
		t.Fatalf("got %q, want %q", result, "a\nd")
	}
}

func TestEditToolOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new.txt")

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "new.txt",
		"edits": []map[string]any{{
			"loc":     "overwrite",
			"content": []string{"line1", "line2", "line3"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("overwrite failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "line1\nline2\nline3" {
		t.Fatalf("got %q", result)
	}
}

func TestEditToolOverwriteExistingFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTestFile(t, dir, "existing.txt", "old content")

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "existing.txt",
		"edits": []map[string]any{{
			"loc":     "overwrite",
			"content": []string{"new content"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("overwrite failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "new content" {
		t.Fatalf("got %q", result)
	}
}

func TestEditToolOverwriteCreatesDir(t *testing.T) {
	dir := t.TempDir()

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "sub/dir/new.txt",
		"edits": []map[string]any{{
			"loc":     "overwrite",
			"content": []string{"hello"},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("overwrite failed: %v", out.Error)
	}

	result := readTestFile(t, filepath.Join(dir, "sub/dir/new.txt"))
	if result != "hello" {
		t.Fatalf("got %q", result)
	}
}

func TestEditToolOverwriteMustBeSoleEdit(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "test.txt", "hello")

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{
			{"loc": "overwrite", "content": []string{"new"}},
			{"loc": "append", "content": []string{"extra"}},
		},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error == nil {
		t.Fatal("expected error when overwrite combined with other edits")
	}
}

func TestEditToolMove(t *testing.T) {
	dir := t.TempDir()
	content := "a\nb\nc\nd\ne"
	p := writeTestFile(t, dir, "test.txt", content)

	wd := &MutableWorkingDir{}
	wd.Set(dir)
	tool := &EditTool{WorkingDir: wd}

	lines := strings.Split(content, "\n")

	input := map[string]any{
		"path": "test.txt",
		"edits": []map[string]any{{
			"loc": map[string]any{
				"move": map[string]string{
					"pos":   tagFor(2, lines[1]),
					"end":   tagFor(3, lines[2]),
					"after": tagFor(4, lines[3]),
				},
			},
		}},
	}
	m, _ := json.Marshal(input)
	out := tool.Run(context.Background(), m)
	if out.Error != nil {
		t.Fatalf("move failed: %v", out.Error)
	}

	result := readTestFile(t, p)
	if result != "a\nd\nb\nc\ne" {
		t.Fatalf("got %q, want %q", result, "a\nd\nb\nc\ne")
	}
}
