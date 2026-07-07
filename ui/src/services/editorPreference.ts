// Shared preference for which local editor the "Open in editor" header button
// targets. Both the header button (ChatInterface.vue) and the toggle in the
// overflow menu (ChatOverflowMenu.vue) import the same module-level ref so a
// change in one place is reflected live in the other, and the choice persists
// across reloads via localStorage.
import { ref } from "vue";
import type { EditorKind } from "./editorUri";

// Re-exported so consumers can import both the pref and the URI builder from
// one place. The URI logic lives in editorUri.ts (no Vue dep) so it stays
// unit-testable under the Node test runner.
export type { EditorKind } from "./editorUri";
export {
  buildRemoteEditorUri,
  buildLocalEditorUri,
  buildEditorUri,
  isLoopbackHost,
} from "./editorUri";

const KEY = "shelley.editor";

// localStorage is guarded so this module is safe to import in environments
// where it is unavailable (e.g. private mode).
function readStored(): EditorKind {
  try {
    const v = typeof localStorage !== "undefined" ? localStorage.getItem(KEY) : null;
    return v === "cursor" ? "cursor" : "vscode";
  } catch {
    return "vscode";
  }
}

export const preferredEditor = ref<EditorKind>(readStored());

export function setPreferredEditor(editor: EditorKind): void {
  preferredEditor.value = editor;
  try {
    if (typeof localStorage !== "undefined") localStorage.setItem(KEY, editor);
  } catch {
    // Ignore storage failures (private mode, quota) — the in-memory ref still
    // reflects the choice for the current session.
  }
}
