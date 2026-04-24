---
name: new-conversation
description: Use when the user asks to "start a new conversation/chat", "launch a new conversation", "spawn a separate chat", or similar. This is NOT the same as a subagent — the user wants an independent conversation they can interact with in the UI, not a silent subtask.
---

When the user asks to start a new, separate Shelley conversation:

1. Do NOT use the `subagent` tool. A subagent is a silent child of this
   conversation; the user wants an independent conversation visible in the
   sidebar.

2. Use the same mechanism as scheduled jobs: shell out to
   `shelley client chat`. Pass the prompt via `-p` and, when relevant, a
   working directory via `-cwd`.

   ```
   shelley client chat -p '<prompt>' -cwd '<working_directory>'
   ```

3. The prompt should convey the user's intent concisely, in their own words
   when possible, and include this conversation's ID (from
   `$SHELLEY_CONVERSATION_ID`) so the new agent can reference context if
   needed.

4. `shelley client chat` returns immediately with the new conversation ID.
   Report that ID back to the user so they can find/open the conversation
   in the UI.

5. Unless the user asked you to wait, do not block on the new conversation's
   output — it runs independently.
