# Molecula → upstream (bold/main) port inventory

Base: `bold/main` @ b31c35e. Our fork: `molecula` (backup branch `molecula-backup`).
Merge-base: `4688882`. 62 molecula commits, 325 upstream commits since.

## PROGRESS — Group A (Go backend) COMPLETE ✅ (15/15)
Ported onto `rebase-onto-upstream` (see `git log 4013a5d..HEAD`). Backend/tooling done;
UI halves of #1/#5/#6/#7/#8/#13 + all of Group B tracked below.

Key reconciliation decisions (differ from a literal molecula replay):
- **#4 edit/patch — COEXIST, not replace.** Added hashline `edit`+`read` tools ALONGSIDE
  upstream's `patch` (which is mature: weak-model schema, UI rendering, predictable.go).
  Reused patch's `PatchDisplayData` shape so edit results render in the existing UI.
  "Remove patch" left as a deliberate cross-cutting follow-up.
- **#13 cost — gateway header PRIMARY, local pricing FALLBACK.** Molecula removed the
  `Exedev-Gateway-Cost` header; we kept it and fall back to local `ModelPrice` tables only
  when it's 0, so cost is right with the gateway AND non-zero with a direct provider key.
- **#5 PR badges** delivered via `PRInfo` on the conversation-list patch stream (upstream's
  architecture), not molecula's separate broadcast/endpoint.
- **#9 skills** — ported only repo-local `.claude/skills/` discovery + `new-conversation`
  builtin. SKIPPED molecula's `skills.go` tree-discovery/always-on refactor + caveman removal:
  upstream's skills system diverged (own builtin set + ListAll/DiscoverInTree) and the
  refactor would regress it. Needs separate reconciliation if still wanted.
- **#14** — moved UI-staleness check out of `ui.init()` into `ui.EnforceFreshBuild()` (called
  only from `serve`); this also unblocked all Go tests that transitively import `ui`.
- **#15 CI** — pointed release/self-update at `molecula/shelley`, dropped homebrew tap, but
  KEPT workflow branch triggers on `main` (this rebased branch becomes the fork's main),
  unlike molecula's dedicated `molecula` branch.

Pre-existing env-only test failures (NOT regressions, exist on bold/main): TestWithAnthropicAPI
(needs API key), TestSystemdListenerIntegration (needs built UI), TestBrowserDownload (needs
headless browser). All ported packages' unit tests pass.

## KEY UPSTREAM CHANGE
Upstream **deleted the React frontend and rewrote the UI in Vue 3 + PrimeVue**.
Every React (.tsx) commit of ours must be **re-implemented in Vue**, not merged.
Upstream also **still uses the `patch` tool** (we replaced it with `edit`/hashline).

## Already in upstream — DO NOT PORT
- Models: claude-opus-4.7, claude-opus-4.8, gpt-5.5 (and more). (commits 3d5187f5, b1383767, f8dc569a, e193213a)
- Brevity system-prompt guidance (be4c14c3, c2dcd4dc) — upstream prompt already says "Communicate with brevity".
- Subagent depth limit backend `MaxSubagentDepth` (backend half of 7a50e9f7).
- Drag-drop file upload **backend** handler (backend half of 0cd5fcfa).
- Notifications dispatcher infra (discord/email/ntfy) — ours built on same infra.
- exe.dev/exe.xyz cleanup (f5b02ca9) — mostly already done upstream; verify no regressions.

---

## FEATURES TO PORT (grouped)

### A. Backend/tooling (Go) — clean adds, low UI coupling
1. **Slack integration** (0c298b04, 732e4bf3, c6735ad7, 8b107850-slack-part)
   - New: `slack/` pkg (Socket Mode client), `claudetool/slack.go` tool, `server/slack_api.go`.
   - Wires: `claudetool.ToolSetConfig.SlackAPI`, `server.SetSlackAPI`, config tokens
     (`slack_bot_token`/`slack_app_token`) in `cmd/shelley/main.go` + `server/llmconfig.go`.
   - Tool actions: send_message, get_history, get_thread, list_channels, add_reaction,
     find_users, lookup_user_by_email.
2. **MCP client** (30a5b87a)
   - New: `mcp/` pkg (generic client + HTTP Streamable transport), `claudetool/mcp_tools.go`
     (deferred tool loading). Wires into `loop/loop.go`, `server/convo.go`, `llmconfig.go`,
     `cmd/shelley/main.go` (mcp_servers config).
3. **RTK bash token optimization** (b51d1817)
   - New: `claudetool/rtk.go`; hook in `claudetool/bash.go`.
4. **Hashline read/edit tools + remove patch** (49db60cf, c0dc0318, adebd824, 680b318d-UI)
   - New: `claudetool/read.go`, `claudetool/edit.go`, `claudetool/hashline/`, `claudetool/editbuf/`.
   - Removes `claudetool/patch.go`/`patchkit`. Updates toolset, prompts, loop/predictable.go,
     models refs, llm packages. **Big & cross-cutting.** UI must render edit results w/ inline diffs.
5. **GitHub PR status badges** (cdb15d98, a9a5bc29, 2911e541, eba292da)
   - New: `gitstate/pr.go`; `server/handlers.go`/`server.go` endpoints; `cmd/go2ts.go` types.
   - UI: PR badges in conversation list (Vue re-impl).
6. **Web push notifications (iOS/macOS)** (7e637a13, 1d814e95)
   - New: `db/push_subscriptions.go`, `db/schema/018-push-subscriptions.sql`,
     `server/notifications/channels/webpush.go`, `server/push_subscriptions.go`. go.mod dep.
   - UI: subscribe/permission flow (Vue re-impl).
7. **Hostname identicon + seashell app icons** (eeb27503)
   - New: `server/host_icon.go`; wires `cmd/shelley/main.go`, `server.go`. UI: show icon.
8. **Slash command palette + skills/commands library** (9ca9e8b8, 135e1f0c)
   - New: `commands/` pkg, `server/commands_handlers.go`. UI: palette (Vue re-impl).
9. **Skills system changes** (aa6ddf6c, bfc3305a, 56ac18e7, 8f22d361)
   - Built-in skills, always-on config, remove tree discovery; discover repo-local
     `.claude/skills/`; add `new-conversation` builtin; remove caveman.
   - NOTE: reconcile with upstream's own builtin skills set. Careful merge.
10. **Config default path** (418d36d5) — default `-config` to `$XDG_CONFIG_HOME/shelley/shelley.json`.
11. **Slug fix** (22ae2706) — skip thinking blocks when models return them.
12. **Anthropic test fix** (c8a29299) — first text content block. (may be obsolete upstream)
13. **Session cost / local pricing** (97fbbb51, 9885c5ac) — DESIGN DIVERGENCE, decide first.
    Upstream computes `CostUSD` from the exe.dev gateway response header
    (`Exedev-Gateway-Cost`, `llm/llm.go:CostUSDFromResponse`). OURS computes cost
    **locally** from token counts + per-model pricing tables (`llm/*/pricing.go`),
    which works without the gateway. Port our local pricing as a fallback when the
    gateway header is absent. UI: session cost in toolbar (Vue re-impl, 97fbbb51 UI half).
14. **Deployment** (b374b189, 94cc8365) — systemd unit, launchd plist, `make install` to
    `~/.local/bin`, failure hook, staleness scope. Reconcile with upstream Makefile.
15. **CI** (90a1a716) — build/self-update from molecula fork; version metadata. Fork-specific.

### B. UI-only — RE-IMPLEMENTED in Vue  (build-verified via `node scripts/build.js`)
Key finding: upstream's Vue rewrite ALREADY had several of these; only genuine gaps were built.
16. Ctrl+C stop-agent shortcut — ✅ DONE (ChatInterface.vue keydown; guards text selection).
17. Cmd+Shift+D directory picker shortcut — ✅ DONE (same handler).
18. Copy-button feedback + clipboard fallback — ✅ DONE (Message.vue now uses shared copyText
    execCommand fallback; feedback already existed upstream).
19. Conversation sort options: activity/created/name — ✅ DONE (sort selector in drawer,
    persisted; SortMode in conversationSort.ts; i18n keys added).
20. Theme picker modal + diff viewer — ✅ ALREADY UPSTREAM (ChatOverflowMenu SelectButton
    system/light/dark; DiffViewer.vue Monaco). No work needed.
21. Nested subagents in sidebar — ✅ ALREADY UPSTREAM (ConversationDrawerRow subagentsByParent).
22. Edit-tool inline diff rendering — ✅ DONE (routed `edit`→PatchTool in MessageContentBlock;
    edit emits same PatchDisplayData shape).

### B2. UI halves of Group-A backend ports (also Vue)
- #7 Host icon — ✅ DONE (HostIcon.vue fetches /api/host-icon, mounted in header).
  (Seashell app-icon PNG/SVG asset swap NOT done — pure branding, optional follow-up.)
- #6 Web push — ✅ DONE (sw.ts service worker + build.js emit; subscribe flow + toggle in
  NotificationsModal).
- #13 Session cost — ✅ DONE (session_cost_usd threaded through messageStore, shown by context bar).
- #5 PR badges — ✅ DONE (badge per conversation row from pr_info).
- #8 Slash palette + Library — dynamic /api/commands merge + Conversations|Library (Skills|Commands) tabs.

### C. Skip / fork-only / trivial
- ci trigger commits (490df9cc, 8322cc93) — empty/no-op.
- Merge commits — N/A.
