1. Never add sleeps to tests.
2. Brevity, brevity, brevity! Do not do weird defaults; have only one way of doing things; refactor relentlessly as necessary.
3. If something doesn't work, propagate the error or exit or crash. Do not have "fallbacks".
4. Do not keep old methods around for "compatibility"; this is a new project and there
   are no compatibility concerns yet.
5. The "predictable" model is a test fixture that lets you specify what a model would say if you said
   a thing. This is useful for interactive testing with a browser, since you don't rely on a model,
   and can fabricate some inputs and outputs. To test things, launch shelley with the relevant flag
   to only expose this model, and use shelley with a browser.
6. Build the UI (`make ui` or `cd ui && pnpm install && pnpm run build`) before running Go tests so `ui/dist` exists for the embed.
7. **Always run `cd ui && pnpm run type-check && pnpm run type-check:vue` after modifying any `.ts`, `.tsx`, or `.vue` file.** Fix all errors before committing. Run linting with `pnpm run lint`.
8. Run Go unit tests with `go test ./server` (or narrower packages while iterating) once the UI bundle is built.
9. To programmatically type into the message input (e.g., in browser automation), set the
   value via the native setter and dispatch an `input` event so Vue's v-model picks it up:
   ```javascript
   const input = document.querySelector('[data-testid="message-input"]');
   const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, "value").set;
   nativeInputValueSetter.call(input, 'your message');
   input.dispatchEvent(new Event('input', { bubbles: true }));
   ```
   Simply setting `input.value = '...'` won't work because the framework won't detect the change.
10. Commit your changes before finishing your turn.
11. If you are testing Shelley itself, be aware that you might be running "under" shelley,
  and indiscriminately running pkill -f shelley may break things.
12. To test the Shelley UI in a separate instance, build with `make build`, then run on a
    different port with a separate database:
    ```
    ./bin/shelley -config /exe.dev/shelley.json -db /tmp/shelley-test.db serve -port 8002
    ```
    Then use browser tools to navigate to http://localhost:8002/ and interact with the UI.
13. NEVER use alert(), confirm(), or prompt(). Use proper UI components like tooltips, modals, or toasts instead.
14. SQL migrations and frontend changes require rebuilding the binary (`make build` or `go generate ./... && cd ui && pnpm run build`).
15. Tool changes and UI tool widget updates go hand in hand. When you add, rename, remove, or
    restructure a tool (in `claudetool/`), you MUST update the UI components that render it:
    - `ui/src/vue/components/CoalescedToolCall.vue` (`TOOL_COMPONENTS` map) and the tool
      components under `ui/src/vue/components/tools/`
    - `ui/src/vue/components/tools/BrowserTool.vue` (if the tool is a browser action — this component
      reads the `action` field from the input and dispatches to the right sub-component)
    - `loop/predictable.go` (the "tool smorgasbord" demo response)

## Deploying to the local systemd instance (safe install/restart)

This host runs Shelley as a systemd **user** service (`systemctl --user … shelley`),
binary at `~/.local/bin/shelley serve`, DB at `~/shelley.db`. To ship changes:

1. **Rebuild first, always.** `make build`. The embedded UI staleness check
   (`ui.EnforceFreshBuild`, runs on `serve`) will exit(1) if any `ui/src` file is
   newer than the embedded `ui/dist` — so an *old* binary won't even start once
   you've edited `ui/src`. Rebuild so `dist` is fresh, then install.
2. **Back up before cutover:** the binary (`cp ~/.local/bin/shelley ~/.local/bin/shelley.bak`)
   and, if any DB migration is involved, the DB (`cp ~/shelley.db ~/shelley.db.bak`).
3. **Install with unlink-then-copy, not plain `cp`.** Overwriting the running binary
   in place fails with `Text file busy` (ETXTBSY). Do `rm -f ~/.local/bin/shelley &&
   cp bin/shelley ~/.local/bin/shelley` (or `mv`, which is what `make install` uses —
   rename replaces the dir entry; the running process keeps its inode until restart).
4. **Restart & verify:** `systemctl --user restart shelley`, then poll
   `curl -s localhost:9000/version` (check the `commit`), and
   `journalctl --user -u shelley.service -n 30` for a clean start (no crash-loop /
   `Main process exited`). A large-DB FTS migration can take ~30–60s before it serves.
5. **Rollback:** restore the binary backup (unlink-then-copy) and, if migrations ran,
   the DB backup; `systemctl --user reset-failed shelley && systemctl --user restart shelley`.

**Migration-lineage gotcha (one-time, when moving this DB onto upstream):** the
`migrations` table has `migration_number` as PRIMARY KEY, and the runner marks
migrations applied **by name**. A DB built on the old fork can hold a different
migration at a number that upstream reuses (e.g. old `018-push-subscriptions` vs
upstream `018-add-reasoning-effort`) → startup dies with
`UNIQUE constraint failed: migrations.migration_number`. Fix by renumbering the
diverged row to its new identity (e.g. push → 034) so the number is freed and the
renamed migration is treated as already-applied. **Always dry-run the reconciliation
on a *copy* of the DB with the new binary on a spare port** (`shelley -db /tmp/copy.db
serve -port 9099 -socket none`) before touching the live DB.

Note: you are usually **not** running under this service (check `pstree -ps $$`); if
you ever are, don't `pkill -f shelley` or restart naively — it'll kill your own turn.
