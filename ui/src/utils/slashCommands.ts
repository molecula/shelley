import { api, type SlashUserCommand } from "../services/api";

export interface SlashCommand {
  command: `/${string}`;
  description: string;
  takesArgs: boolean;
  // Present only for user-defined commands loaded from /api/commands. When set,
  // selecting the command inserts the rendered body (with $ARGUMENTS handling)
  // into the composer rather than sending `/name` verbatim. Built-ins leave
  // these undefined and keep their existing send/insert behavior.
  isUserCommand?: boolean;
  body?: string;
  argumentHint?: string;
}

export const SLASH_COMMANDS = {
  FORK: {
    command: "/fork",
    description: "forks this conversation",
    takesArgs: false,
  },
  DIFF: {
    command: "/diff",
    description: "opens the diff viewer",
    takesArgs: false,
  },
  SHELL: {
    command: "/shell",
    description: "runs in shell (! alias)",
    takesArgs: true,
  },
  COMPACT: {
    command: "/compact",
    description: "compacts this conversation",
    takesArgs: true,
  },
  DISTILL: {
    // Legacy alias for /compact, kept for compatibility. Compacts too.
    command: "/distill",
    description: "compacts this conversation (alias for /compact)",
    takesArgs: true,
  },
  NEW: {
    command: "/new",
    description: "starts a new conversation",
    takesArgs: true,
  },
  ARCHIVE: {
    command: "/archive",
    description: "archives this conversation",
    takesArgs: false,
  },
} as const satisfies Record<string, SlashCommand>;

// Names of the hardcoded built-ins, used to dedupe against user commands.
// Built-ins win: a user command whose name collides with a built-in is
// dropped, so the special UI actions (/fork, /compact, …) always behave.
const BUILTIN_NAMES = new Set(
  Object.values(SLASH_COMMANDS).map((c) => c.command.slice(1).toLowerCase()),
);

// toSlashCommand adapts an /api/commands user command into the SlashCommand
// shape the composer menu already renders. A command "takes args" when its
// body has an $ARGUMENTS placeholder or it declares an argument-hint.
function toSlashCommand(c: SlashUserCommand): SlashCommand {
  const takesArgs = c.body.includes("$ARGUMENTS") || !!c.argument_hint;
  return {
    command: `/${c.name}`,
    description: c.description || c.path,
    takesArgs,
    isUserCommand: true,
    body: c.body,
    argumentHint: c.argument_hint,
  };
}

// fetchUserSlashCommands loads user-defined commands from /api/commands and
// maps them to SlashCommand entries, skipping any whose name collides with a
// hardcoded built-in. Callers merge these alongside SLASH_COMMANDS. Failures
// resolve to an empty list so the composer degrades to built-ins only.
export async function fetchUserSlashCommands(cwd?: string): Promise<SlashCommand[]> {
  try {
    const data = await api.getCommands(cwd);
    return (data.user_commands ?? [])
      .filter((c) => !BUILTIN_NAMES.has(c.name.toLowerCase()))
      .map(toSlashCommand);
  } catch {
    return [];
  }
}

// renderUserCommand substitutes $ARGUMENTS in a command body with the given
// arguments; if the body has no placeholder, args are appended on a new line.
// Mirrors commands.Render on the server. Uses a function replacement so that
// `$`-sequences inside args aren't reinterpreted by String.replace.
export function renderUserCommand(body: string, args: string): string {
  if (body.includes("$ARGUMENTS")) {
    return body.replace(/\$ARGUMENTS/g, () => args);
  }
  return args ? `${body}\n\n${args}` : body;
}
