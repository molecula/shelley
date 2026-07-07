import {
  buildRemoteEditorUri,
  buildLocalEditorUri,
  buildEditorUri,
  isLoopbackHost,
} from "./editorUri";

function assert(cond: boolean, msg: string): void {
  if (!cond) throw new Error(`Assertion failed: ${msg}`);
}

function run(name: string, fn: () => void): void {
  try {
    fn();
    console.log(`✓ ${name}`);
  } catch (err) {
    console.error(`✗ ${name}`);
    throw err;
  }
}

run("builds a vscode Remote-SSH URI from host + absolute cwd", () => {
  const uri = buildRemoteEditorUri("vscode", "jaffee-pig", "/home/danish/circuit-2026-07-07");
  assert(
    uri === "vscode://vscode-remote/ssh-remote+jaffee-pig/home/danish/circuit-2026-07-07",
    `got: ${uri}`,
  );
});

run("builds a cursor Remote-SSH URI with the cursor scheme", () => {
  const uri = buildRemoteEditorUri("cursor", "jaffee-pig", "/home/danish/circuit-2026-07-07");
  assert(
    uri === "cursor://vscode-remote/ssh-remote+jaffee-pig/home/danish/circuit-2026-07-07",
    `got: ${uri}`,
  );
});

run("escapes unsafe path characters while preserving slashes", () => {
  const uri = buildRemoteEditorUri("vscode", "box", "/home/me/my repo-2026");
  assert(uri === "vscode://vscode-remote/ssh-remote+box/home/me/my%20repo-2026", `got: ${uri}`);
});

run("builds a local vscode file URI (no Remote-SSH)", () => {
  const uri = buildLocalEditorUri("vscode", "/home/danish/circuit-2026-07-07");
  assert(uri === "vscode://file/home/danish/circuit-2026-07-07", `got: ${uri}`);
});

run("builds a local cursor file URI with the cursor scheme", () => {
  const uri = buildLocalEditorUri("cursor", "/home/danish/circuit-2026-07-07");
  assert(uri === "cursor://file/home/danish/circuit-2026-07-07", `got: ${uri}`);
});

run("local file URI escapes unsafe path characters while preserving slashes", () => {
  const uri = buildLocalEditorUri("vscode", "/home/me/my repo-2026");
  assert(uri === "vscode://file/home/me/my%20repo-2026", `got: ${uri}`);
});

run("recognizes loopback hosts", () => {
  assert(isLoopbackHost("localhost"), "localhost");
  assert(isLoopbackHost("127.0.0.1"), "127.0.0.1");
  assert(isLoopbackHost("::1"), "::1");
  assert(isLoopbackHost("[::1]"), "[::1]");
});

run("does not treat a remote hostname as loopback", () => {
  assert(!isLoopbackHost("jaffee-pig"), "jaffee-pig");
  assert(!isLoopbackHost("localhost.example.com"), "localhost.example.com");
});

run("buildEditorUri uses a local file URI when served over loopback", () => {
  const uri = buildEditorUri("vscode", {
    locationHost: "localhost",
    sshHost: "jaffee-pig",
    path: "/home/danish/circuit",
  });
  assert(uri === "vscode://file/home/danish/circuit", `got: ${uri}`);
});

run("buildEditorUri uses Remote-SSH when served over a remote hostname", () => {
  const uri = buildEditorUri("cursor", {
    locationHost: "jaffee-pig",
    sshHost: "jaffee-pig",
    path: "/home/danish/circuit",
  });
  assert(
    uri === "cursor://vscode-remote/ssh-remote+jaffee-pig/home/danish/circuit",
    `got: ${uri}`,
  );
});

console.log("\neditorPreference tests passed");
