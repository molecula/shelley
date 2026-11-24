package hashline

import (
	"fmt"
	"strings"
	"testing"
)

// at returns an anchor for 1-indexed line n of text.
func at(text string, n int) *Anchor {
	lines := strings.Split(text, "\n")
	return &Anchor{Line: n, Hash: ComputeLineHash(n, lines[n-1])}
}

const five = "a\nb\nc\nd\ne"

func TestApplyEditsSameLineCombinations(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		edits []Edit
		want  string
	}{
		{
			name: "delete a line and insert after it",
			text: five,
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 2)},
				{Op: "append_at", Pos: at(five, 2), Content: []string{"X"}},
			},
			want: "a\nX\nc\nd\ne",
		},
		{
			name: "grow a line and insert after it",
			text: five,
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 2), Content: []string{"B1", "B2", "B3"}},
				{Op: "append_at", Pos: at(five, 2), Content: []string{"X"}},
			},
			want: "a\nB1\nB2\nB3\nX\nc\nd\ne",
		},
		{
			name: "delete the whole file and insert after its last line",
			text: five,
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 1), End: at(five, 5)},
				{Op: "append_at", Pos: at(five, 5), Content: []string{"X"}},
			},
			want: "X",
		},
		{
			name: "replace a range and insert before its first line",
			text: five,
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 4), Content: []string{"M"}},
				{Op: "prepend_at", Pos: at(five, 2), Content: []string{"X"}},
			},
			want: "a\nX\nM\ne",
		},
		{
			name: "insert between two adjacent replaced ranges",
			text: five,
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 1), End: at(five, 2), Content: []string{"P"}},
				{Op: "replace_range", Pos: at(five, 3), End: at(five, 4), Content: []string{"Q"}},
				{Op: "append_at", Pos: at(five, 2), Content: []string{"X"}},
			},
			want: "P\nX\nQ\ne",
		},
		{
			name: "two moves in one call",
			text: five,
			edits: []Edit{
				{Op: "move_range", Pos: at(five, 1), End: at(five, 1), After: at(five, 3)},
				{Op: "move_range", Pos: at(five, 2), End: at(five, 2), After: at(five, 5)},
			},
			want: "c\na\nd\ne\nb",
		},
		{
			name: "inserts at one anchor keep the order they were given in",
			text: five,
			edits: []Edit{
				{Op: "append_at", Pos: at(five, 2), Content: []string{"X"}},
				{Op: "append_at", Pos: at(five, 2), Content: []string{"Y"}},
			},
			want: "a\nb\nX\nY\nc\nd\ne",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEdits(tt.text, tt.edits)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyEditsConflictsAreRejected(t *testing.T) {
	tests := []struct {
		name  string
		edits []Edit
	}{
		{
			name: "overlapping ranges",
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 1), End: at(five, 3), Content: []string{"P"}},
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 5), Content: []string{"Q"}},
			},
		},
		{
			name: "same line replaced twice",
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 2), Content: []string{"P"}},
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 2), Content: []string{"Q"}},
			},
		},
		{
			name: "insert after a line inside a deleted range",
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 1), End: at(five, 5)},
				{Op: "append_at", Pos: at(five, 4), Content: []string{"X"}},
			},
		},
		{
			name: "insert before a line inside a replaced range",
			edits: []Edit{
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 4), Content: []string{"M"}},
				{Op: "prepend_at", Pos: at(five, 3), Content: []string{"X"}},
			},
		},
		{
			name: "move a range that another edit replaces",
			edits: []Edit{
				{Op: "move_range", Pos: at(five, 1), End: at(five, 2), After: at(five, 5)},
				{Op: "replace_range", Pos: at(five, 2), End: at(five, 3), Content: []string{"Q"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEdits(five, tt.edits)
			if err == nil {
				t.Fatalf("expected a conflict error, got %q", got)
			}
			if !strings.Contains(err.Error(), "conflicting edits") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestApplyEditsAppendFileKeepsTrailingNewline(t *testing.T) {
	got, err := ApplyEdits("a\nb\n", []Edit{{Op: "append_file", Content: []string{"c"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb\nc\n" {
		t.Fatalf("got %q", got)
	}

	// Content inserted before the final empty line stays above the append.
	got, err = ApplyEdits("a\n", []Edit{
		{Op: "append_file", Content: []string{"C"}},
		{Op: "prepend_at", Pos: at("a\n", 2), Content: []string{"B"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nB\nC\n" {
		t.Fatalf("got %q", got)
	}

	// No trailing newline: content goes on a new last line, as before.
	got, err = ApplyEdits("a\nb", []Edit{{Op: "append_file", Content: []string{"c"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb\nc" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTagErrorExampleIsValid(t *testing.T) {
	_, err := ParseTag("not a tag")
	if err == nil {
		t.Fatal("expected an error")
	}
	// The example the error offers the model must itself parse.
	msg := err.Error()
	start := strings.Index(msg, `e.g. "`)
	if start < 0 {
		t.Fatalf("no example in error: %s", msg)
	}
	example := msg[start+len(`e.g. "`):]
	example = example[:strings.Index(example, `"`)]
	if _, err := ParseTag(example); err != nil {
		t.Fatalf("example %q from the error message does not parse: %v", example, err)
	}
}

// FuzzApplyEdits checks that no batch of validly anchored edits can panic,
// that a batch is rejected exactly when it conflicts, and that an accepted
// batch produces exactly the output of model, an independent implementation.
func FuzzApplyEdits(f *testing.F) {
	f.Add(5, false, []byte{0, 1, 3, 0, 2, 5, 0, 0})
	f.Add(5, true, []byte{1, 1, 5, 0, 2, 5, 0, 0, 4, 0, 0, 0})
	f.Add(3, true, []byte{3, 3, 3, 0, 4, 0, 0, 0, 2, 3, 0, 0})
	f.Add(6, false, []byte{6, 0, 1, 4, 6, 2, 2, 5, 2, 1, 0, 0})
	f.Fuzz(func(t *testing.T, n int, trailingNewline bool, script []byte) {
		if n < 1 || n > 12 || len(script) > 32 {
			t.Skip()
		}
		lines := make([]string, n)
		for i := range lines {
			lines[i] = string(rune('a' + i))
		}
		text := strings.Join(lines, "\n")
		if trailingNewline {
			text += "\n"
		}
		count := len(strings.Split(text, "\n"))
		anchor := func(b byte) *Anchor { return at(text, int(b)%count+1) }

		var edits []Edit
		for i := 0; i+3 < len(script); i += 4 {
			a, b := anchor(script[i+1]), anchor(script[i+2])
			if a.Line > b.Line {
				a, b = b, a
			}
			tag := "N" + string(rune('0'+i/4))
			content := []string{tag}
			switch script[i] % 8 {
			case 0:
				edits = append(edits, Edit{Op: "replace_range", Pos: a, End: b, Content: content})
			case 1:
				edits = append(edits, Edit{Op: "replace_range", Pos: a, End: b})
			case 2:
				edits = append(edits, Edit{Op: "append_at", Pos: a, Content: content})
			case 3:
				edits = append(edits, Edit{Op: "prepend_at", Pos: a, Content: content})
			case 4:
				edits = append(edits, Edit{Op: "append_file", Content: content})
			case 5:
				edits = append(edits, Edit{Op: "prepend_file", Content: content})
			case 6:
				edits = append(edits, Edit{Op: "move_range", Pos: a, End: b, After: anchor(script[i+3])})
			case 7:
				edits = append(edits, Edit{Op: "replace_range", Pos: a, End: b, Content: []string{tag + "a", tag + "b"}})
			}
		}

		got, err := ApplyEdits(text, edits) // must not panic
		want, ok := model(text, edits)
		switch {
		case err != nil && ok:
			t.Fatalf("rejected a valid batch: %v (edits %s, model %q)", err, describe(edits), want)
		case err == nil && !ok:
			t.Fatalf("accepted a conflicting batch (edits %s, result %q)", describe(edits), got)
		case err == nil && got != want:
			t.Fatalf("got %q, want %q (text %q, edits %s)", got, want, text, describe(edits))
		}
	})
}

func describe(edits []Edit) string {
	var b strings.Builder
	line := func(a *Anchor) int {
		if a == nil {
			return 0
		}
		return a.Line
	}
	for _, e := range edits {
		fmt.Fprintf(&b, "[%s %d-%d after=%d %q] ", e.Op, line(e.Pos), line(e.End), line(e.After), e.Content)
	}
	return b.String()
}

// model applies edits one at a time to a list of items, finding each anchor by
// the identity of the original line it names rather than by position. A
// replaced range leaves start and end markers behind, so later inserts can
// still be anchored at its edges. It reports false for a batch that must be
// rejected: overlapping ranges, an insert anchored inside a range, or a move
// into its own source.
func model(text string, edits []Edit) (string, bool) {
	type item struct {
		line int    // original line number, for an original line or a marker
		kind byte   // 'o' original line, '<' range start, '>' range end, 't' inserted text
		text string // the line's text
	}
	fileLines := strings.Split(text, "\n")
	var items []item
	for i, l := range fileLines {
		items = append(items, item{line: i + 1, kind: 'o', text: l})
	}
	find := func(kind byte, line int) int {
		for i, it := range items {
			if it.kind == kind && it.line == line {
				return i
			}
		}
		return -1
	}
	insert := func(at int, lines []string) {
		var add []item
		for _, l := range lines {
			add = append(add, item{kind: 't', text: l})
		}
		items = append(items[:at], append(add, items[at:]...)...)
	}

	// A move is a delete of its source plus an insert of the same lines after
	// its destination.
	var expanded []Edit
	for _, e := range edits {
		if e.Op != "move_range" {
			expanded = append(expanded, e)
			continue
		}
		if e.After.Line >= e.Pos.Line && e.After.Line <= e.End.Line {
			return "", false
		}
		expanded = append(expanded,
			Edit{Op: "replace_range", Pos: e.Pos, End: e.End},
			Edit{Op: "append_at", Pos: e.After, Content: fileLines[e.Pos.Line-1 : e.End.Line]})
	}

	// Ranges first. Every original line in the range must still be present.
	for _, e := range expanded {
		if e.Op != "replace_range" {
			continue
		}
		for l := e.Pos.Line; l <= e.End.Line; l++ {
			if find('o', l) < 0 {
				return "", false
			}
		}
		start, end := find('o', e.Pos.Line), find('o', e.End.Line)
		repl := []item{{line: e.Pos.Line, kind: '<'}}
		for _, l := range e.Content {
			repl = append(repl, item{kind: 't', text: l})
		}
		repl = append(repl, item{line: e.End.Line, kind: '>'})
		items = append(items[:start], append(repl, items[end+1:]...)...)
	}

	// If the file ends in a newline and its final empty line survives, appended
	// content goes just before that line, after anything prepended to it;
	// otherwise it goes at the very end.
	last := len(fileLines)
	keepsNewline := last > 1 && fileLines[last-1] == "" && find('o', last) >= 0
	var appendFile, prependFile []string
	for _, e := range expanded {
		switch e.Op {
		case "append_file":
			appendFile = append(appendFile, e.Content...)
		case "prepend_file":
			prependFile = append(prependFile, e.Content...)
		}
	}
	// Each prepend goes immediately before its anchor, so prepends at one
	// anchor keep their order.
	for _, e := range expanded {
		if e.Op != "prepend_at" {
			continue
		}
		i := find('o', e.Pos.Line)
		if i < 0 {
			i = find('<', e.Pos.Line)
		}
		if i < 0 {
			return "", false
		}
		insert(i, e.Content)
	}

	// Appended content goes after anything prepended to the final empty line.
	if keepsNewline {
		insert(find('o', last), appendFile)
	}

	// Each append goes immediately after its anchor; applying them in reverse
	// keeps appends at one anchor in their order.
	for k := len(expanded) - 1; k >= 0; k-- {
		e := expanded[k]
		if e.Op != "append_at" {
			continue
		}
		i := find('o', e.Pos.Line)
		if i < 0 {
			i = find('>', e.Pos.Line)
		}
		if i < 0 {
			return "", false
		}
		insert(i+1, e.Content)
	}

	insert(0, prependFile)
	if !keepsNewline {
		insert(len(items), appendFile)
	}

	var out []string
	for _, it := range items {
		if it.kind == 'o' || it.kind == 't' {
			out = append(out, it.text)
		}
	}
	return strings.Join(out, "\n"), true
}
