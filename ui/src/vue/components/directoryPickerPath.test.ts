import {
  computeTabCompletion,
  parseInputPath,
  resolveSelectedPath,
  type DirEntryLike,
} from "./directoryPickerPath";

let passed = 0;
let failed = 0;

function eq(actual: unknown, expected: unknown, msg: string) {
  const a = JSON.stringify(actual);
  const e = JSON.stringify(expected);
  if (a === e) {
    passed++;
  } else {
    failed++;
    console.error(`✗ ${msg}\n    expected ${e}\n    got      ${a}`);
  }
}

const dir = (name: string): DirEntryLike => ({ name, is_dir: true });
const file = (name: string): DirEntryLike => ({ name, is_dir: false });

// --- parseInputPath ---
eq(parseInputPath(""), { dirPath: "", prefix: "" }, "empty");
eq(parseInputPath("/foo/"), { dirPath: "/foo", prefix: "" }, "trailing slash");
eq(parseInputPath("/foo/bar"), { dirPath: "/foo", prefix: "bar" }, "prefix");
eq(parseInputPath("/foo"), { dirPath: "/", prefix: "foo" }, "root-level prefix");
eq(parseInputPath("/"), { dirPath: "/", prefix: "" }, "root");

// --- resolveSelectedPath: the BUG fix ---
const entries = [dir("projects"), dir("proj"), file("readme.txt")];
// Exact existing dir typed without trailing slash -> select THAT dir, not parent.
eq(
  resolveSelectedPath("/home/projects", "/home", entries, "/home"),
  "/home/projects",
  "exact dir honored (bug fix)",
);
// Trailing slash means we're inside it.
eq(
  resolveSelectedPath("/home/projects/", "/home/projects", [], "/home/projects"),
  "/home/projects",
  "trailing slash selects dir itself",
);
// Non-matching prefix falls back to parent dir listing.
eq(
  resolveSelectedPath("/home/xyz", "/home", entries, "/home"),
  "/home",
  "non-existent segment falls back to parent",
);
// Root-level exact dir.
eq(
  resolveSelectedPath("/projects", "/", [dir("projects")], "/"),
  "/projects",
  "root-level exact dir honored",
);

// --- computeTabCompletion ---
const es = [dir("apple"), dir("application"), dir("banana"), file("apex.txt")];
// Longest common prefix of dirs starting with "app" -> "appl"
eq(
  computeTabCompletion("/x", "app", es),
  "/x/appl",
  "LCP completion (ignores files)",
);
// Single match completes fully with trailing slash.
eq(computeTabCompletion("/x", "ban", es), "/x/banana/", "single match full complete");
// Already at ambiguity point -> null (no change).
eq(computeTabCompletion("/x", "appl", es), null, "at ambiguity point -> null");
// No matches -> null.
eq(computeTabCompletion("/x", "zzz", es), null, "no matches -> null");
// Root base.
eq(computeTabCompletion("/", "ban", es), "/banana/", "root base join");
// Empty prefix with a single dir completes it.
eq(computeTabCompletion("/x", "", [dir("only")]), "/x/only/", "empty prefix single dir");

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
