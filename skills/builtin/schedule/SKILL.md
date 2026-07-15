---
name: schedule
description: Use when a user requests a task to be done later or on a schedule.
---

Use systemd user timer units. Unless the user explicitly asked to schedule something, have the user confirm.

If the task is obvious and straightforward, such as a bash command, you might schedule just the command. Otherwise, schedule a future Shelley conversation by calling `shelley client chat`.

Name shelley-calling units `shelley-<name>.{service,timer}`. The service ExecStart should invoke:

```
shelley client chat -p '<prompt>' -cwd '<working_directory>' -schedule-name 'shelley-<name>'
```

The `-schedule-name` flag (value must match the unit base name, `shelley-<name>`) makes the client append a run record — timestamp, conversation ID, and slug — to `~/.config/shelley/runs/shelley-<name>.jsonl` each time it fires. The Scheduled Tasks UI reads this log to show each task's run history and link to the conversation every run created. Always pass it for shelley-calling units.

After writing the unit files, you MUST activate the timer, otherwise it stays inactive and never fires:

```
systemctl --user daemon-reload
systemctl --user enable --now shelley-<name>.timer
```

`enable --now` both enables the timer and starts it immediately, so it becomes active in the current session. If this Shelley instance runs headless (no interactive login session — common for a long-running server), user timers only fire while a user session is present, so also enable lingering once: `loginctl enable-linger "$USER"`. Verify the timer is armed with `systemctl --user list-timers 'shelley-*'` (it should list a NEXT elapse time); if NEXT is blank or the unit is inactive, it will not run.

Each timer firing always creates a new conversation so that no conversation grows without bound.

The prompt baked into the service unit should concisely convey the overarching goals and context from the user, preferably in their own words, as well as the specific task being achieved by this scheduled invocation. The prompt must always include the originating conversation ID (from `$SHELLEY_CONVERSATION_ID`), so that new agent can refer to the originating conversation for additional context if needed.

You are responsible for ensuring that one-shot units will be cleaned up.
