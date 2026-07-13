// Pure path helpers for DirectoryPickerModal.vue, extracted for unit testing.

export interface DirEntryLike {
  name: string;
  is_dir: boolean;
}

/** Split raw input into the directory to list and the trailing filter prefix. */
export function parseInputPath(path: string): { dirPath: string; prefix: string } {
  if (!path) return { dirPath: "", prefix: "" };
  if (path.endsWith("/")) return { dirPath: path.slice(0, -1) || "/", prefix: "" };
  const lastSlash = path.lastIndexOf("/");
  if (lastSlash === -1) return { dirPath: "", prefix: path };
  if (lastSlash === 0) return { dirPath: "/", prefix: path.slice(1) };
  return { dirPath: path.slice(0, lastSlash), prefix: path.slice(lastSlash + 1) };
}

/** Join a base directory and a child name, handling the root ("/") case. */
export function joinPath(base: string, name: string): string {
  return base === "/" ? `/${name}` : `${base}/${name}`;
}

/**
 * Bash-style tab completion. Given the current (server-normalized) base dir, the
 * typed filter prefix, and the dir's entries, return the new input value or null
 * if nothing should change. Completes to the longest common prefix of matching
 * directories; a single match completes fully (with trailing slash).
 */
export function computeTabCompletion(
  base: string,
  prefix: string,
  entries: DirEntryLike[],
): string | null {
  const matches = entries.filter((e) => e.is_dir && e.name.startsWith(prefix));
  if (matches.length === 0) return null;

  let lcp = matches[0].name;
  for (const m of matches) {
    while (!m.name.startsWith(lcp)) lcp = lcp.slice(0, -1);
  }
  // Already at the ambiguity point: nothing more to complete.
  if (lcp === prefix && matches.length > 1) return null;

  let completed = joinPath(base, lcp);
  if (matches.length === 1) completed += "/";
  return completed;
}

/**
 * Resolve the path to select from the raw input. If the input has no trailing
 * slash and its last segment names an existing directory, honor it exactly
 * instead of falling back to the parent dir.
 */
export function resolveSelectedPath(
  raw: string,
  base: string,
  entries: DirEntryLike[],
  fallback: string,
): string {
  const { dirPath, prefix } = parseInputPath(raw);
  if (!raw.endsWith("/") && prefix) {
    const match = entries.find((e) => e.is_dir && e.name === prefix);
    if (match) return joinPath(base, prefix);
  }
  const selectedPath = raw.endsWith("/") ? (dirPath === "/" ? "/" : dirPath) : dirPath;
  return selectedPath || fallback;
}
