package hashline

import (
	"fmt"
	"strings"
)

// Edit represents a single hashline edit operation.
type Edit struct {
	Op      string   // "replace_range", "append_at", "prepend_at", "append_file", "prepend_file", "move_range"
	Pos     *Anchor  // start anchor (nil for file-level ops)
	End     *Anchor  // end anchor (only for replace_range and move_range)
	After   *Anchor  // destination anchor (only for move_range)
	Content []string // replacement/inserted lines; nil means delete
}

// ApplyEdits validates all anchors and applies edits to file content.
// Every anchor refers to the original file. Edits that touch the same lines
// are rejected rather than applied in an order the caller cannot predict.
func ApplyEdits(text string, edits []Edit) (string, error) {
	if len(edits) == 0 {
		return text, nil
	}

	fileLines := strings.Split(text, "\n")

	// Pre-validate: collect all hash mismatches before mutating.
	var mismatches []HashMismatch
	validateRef := func(ref *Anchor) {
		if ref == nil {
			return
		}
		if ref.Line < 1 || ref.Line > len(fileLines) {
			// Out-of-range is a hard error, not a mismatch.
			return
		}
		actual := ComputeLineHash(ref.Line, fileLines[ref.Line-1])
		if actual != ref.Hash {
			mismatches = append(mismatches, HashMismatch{Line: ref.Line, Expected: ref.Hash, Actual: actual})
		}
	}

	for _, edit := range edits {
		switch edit.Op {
		case "replace_range":
			if edit.Pos == nil || edit.End == nil {
				return "", fmt.Errorf("replace_range requires both pos and end anchors")
			}
			if edit.Pos.Line > edit.End.Line {
				return "", fmt.Errorf("range start line %d must be <= end line %d", edit.Pos.Line, edit.End.Line)
			}
			validateRef(edit.Pos)
			validateRef(edit.End)
		case "move_range":
			if edit.Pos == nil || edit.End == nil || edit.After == nil {
				return "", fmt.Errorf("move_range requires pos, end, and after anchors")
			}
			if edit.Pos.Line > edit.End.Line {
				return "", fmt.Errorf("range start line %d must be <= end line %d", edit.Pos.Line, edit.End.Line)
			}
			if edit.After.Line >= edit.Pos.Line && edit.After.Line <= edit.End.Line {
				return "", fmt.Errorf("move_range destination line %d must not be within source range [%d, %d]", edit.After.Line, edit.Pos.Line, edit.End.Line)
			}
			validateRef(edit.Pos)
			validateRef(edit.End)
			validateRef(edit.After)
		case "append_at", "prepend_at":
			if edit.Pos == nil {
				return "", fmt.Errorf("%s requires pos anchor", edit.Op)
			}
			validateRef(edit.Pos)
		case "append_file", "prepend_file":
			// No anchor needed.
		default:
			return "", fmt.Errorf("unknown edit operation %q", edit.Op)
		}

		// Range check.
		if edit.Pos != nil && (edit.Pos.Line < 1 || edit.Pos.Line > len(fileLines)) {
			return "", fmt.Errorf("line %d does not exist (file has %d lines)", edit.Pos.Line, len(fileLines))
		}
		if edit.End != nil && (edit.End.Line < 1 || edit.End.Line > len(fileLines)) {
			return "", fmt.Errorf("line %d does not exist (file has %d lines)", edit.End.Line, len(fileLines))
		}
		if edit.After != nil && (edit.After.Line < 1 || edit.After.Line > len(fileLines)) {
			return "", fmt.Errorf("line %d does not exist (file has %d lines)", edit.After.Line, len(fileLines))
		}
	}

	if len(mismatches) > 0 {
		return "", &HashMismatchError{Mismatches: mismatches, FileLines: fileLines}
	}

	// Expand move_range into delete + insert.
	var expanded []Edit
	for _, edit := range edits {
		if edit.Op == "move_range" {
			// Extract content from source.
			content := make([]string, edit.End.Line-edit.Pos.Line+1)
			copy(content, fileLines[edit.Pos.Line-1:edit.End.Line])
			// Delete source.
			expanded = append(expanded, Edit{Op: "replace_range", Pos: edit.Pos, End: edit.End, Content: nil})
			// Insert at destination.
			expanded = append(expanded, Edit{Op: "append_at", Pos: edit.After, Content: content})
		} else {
			expanded = append(expanded, edit)
		}
	}
	edits = expanded

	// Index every edit by the original line it is anchored to. Nothing is
	// mutated in place: the output is rebuilt in one pass over the original
	// lines, so no edit can shift the line numbers another edit relies on.
	n := len(fileLines)
	ranges := make(map[int]lineRange) // start line -> range
	ends := make(map[int]bool)        // ends[L]: a range ends at line L
	covered := make([]bool, n+2)      // covered[L]: line L is inside a replace_range
	before := make(map[int][]string)  // lines inserted immediately before line L
	after := make(map[int][]string)   // lines inserted immediately after line L
	var prependFile, appendFile []string

	for _, e := range edits {
		if e.Op != "replace_range" {
			continue
		}
		for l := e.Pos.Line; l <= e.End.Line; l++ {
			if covered[l] {
				return "", fmt.Errorf("conflicting edits: line %d is replaced or moved by more than one edit; combine them into a single range", l)
			}
			covered[l] = true
		}
		ranges[e.Pos.Line] = lineRange{end: e.End.Line, content: e.Content}
		ends[e.End.Line] = true
	}

	for _, e := range edits {
		switch e.Op {
		case "append_at":
			// The gap after line L is inside a range when both L and L+1 are covered.
			if covered[e.Pos.Line] && covered[e.Pos.Line+1] && !ends[e.Pos.Line] {
				return "", fmt.Errorf("conflicting edits: cannot insert after line %d because it is inside a range that is replaced or moved; anchor the insert at the end of that range instead", e.Pos.Line)
			}
			after[e.Pos.Line] = append(after[e.Pos.Line], e.Content...)
		case "prepend_at":
			if _, starts := ranges[e.Pos.Line]; covered[e.Pos.Line] && !starts {
				return "", fmt.Errorf("conflicting edits: cannot insert before line %d because it is inside a range that is replaced or moved; anchor the insert at the start of that range instead", e.Pos.Line)
			}
			before[e.Pos.Line] = append(before[e.Pos.Line], e.Content...)
		case "append_file":
			appendFile = append(appendFile, e.Content...)
		case "prepend_file":
			prependFile = append(prependFile, e.Content...)
		}
	}

	// A file that ends in a newline splits into a final empty element. Content
	// appended to the file belongs just before that element, after anything
	// inserted before it, so the file keeps its single trailing newline
	// instead of gaining a blank line and losing it.
	endsWithNewline := n > 1 && fileLines[n-1] == "" && !covered[n]

	out := make([]string, 0, n+len(prependFile)+len(appendFile))
	out = append(out, prependFile...)
	for l := 1; l <= n; l++ {
		out = append(out, before[l]...)
		if l == n && endsWithNewline {
			out = append(out, appendFile...)
			appendFile = nil
		}
		if r, ok := ranges[l]; ok {
			out = append(out, r.content...)
			l = r.end
		} else {
			out = append(out, fileLines[l-1])
		}
		out = append(out, after[l]...)
	}
	out = append(out, appendFile...)

	return strings.Join(out, "\n"), nil
}

// lineRange is a replace_range keyed by its start line.
type lineRange struct {
	end     int
	content []string
}
