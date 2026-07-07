// Pure, dependency-free editor deep-link logic. Kept separate from
// editorPreference.ts (which pulls in Vue's `ref`) so it can be imported under
// the Node test runner (scripts/run-tests.mjs) without resolving `vue`.

export type EditorKind = "vscode" | "cursor";

// Build the editor deep link that opens a folder on a remote host over
// Remote-SSH. VSCode and Cursor (a VSCode fork) share the same vscode-remote
// URL-handler path; only the scheme differs. `path` is an absolute POSIX path
// (the conversation's cwd) that already begins with "/"; encodeURI keeps the
// slashes while escaping spaces and other unsafe characters.
export function buildRemoteEditorUri(editor: EditorKind, host: string, path: string): string {
  return `${editor}://vscode-remote/ssh-remote+${host}${encodeURI(path)}`;
}

// Build the editor deep link that opens a folder on the *local* machine (no
// Remote-SSH). Used when the browser is on the same host as the Shelley server,
// so the editor can open the path directly. `path` is an absolute POSIX path
// beginning with "/"; encodeURI keeps the slashes while escaping unsafe chars.
export function buildLocalEditorUri(editor: EditorKind, path: string): string {
  return `${editor}://file${encodeURI(path)}`;
}

// True when `host` is a loopback address, i.e. the browser is on the same
// machine as the Shelley server. In that case the worktree lives on the same
// box the editor runs on, so a local file URI works and Remote-SSH (which would
// try to `ssh` back into the very machine we're on) must be avoided.
export function isLoopbackHost(host: string): boolean {
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host === "[::1]";
}

// Pick the correct editor deep link for the current access context: a local
// file URI when the browser reaches Shelley over loopback, otherwise a
// Remote-SSH URI keyed on the server's bare hostname.
export function buildEditorUri(
  editor: EditorKind,
  opts: { locationHost: string; sshHost: string; path: string },
): string {
  if (isLoopbackHost(opts.locationHost)) {
    return buildLocalEditorUri(editor, opts.path);
  }
  return buildRemoteEditorUri(editor, opts.sshHost, opts.path);
}
