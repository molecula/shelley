import { api, type SlashSkill, type SlashUserCommand } from "../services/api";

export interface SlashCommand {
  command: `/${string}`;
  description: string;
  takesArgs: boolean;
  // Present only for user-defined commands loaded from /api/commands. When set,
  // selecting the command inserts a "Use the command X located at <path>"
  // directive into the composer (see renderCommandDirective), pointing the
  // agent at the command file rather than pasting its body. Built-ins leave
  // these undefined and keep their existing send/insert behavior.
  isUserCommand?: boolean;
  commandPath?: string;
  // Present only for skills loaded from /api/commands. When set, selecting the
  // entry inserts a "Use the skill X located at <path>" directive into the
  // composer. skillPath is empty for built-in skills (no filesystem path).
  isSkill?: boolean;
  skillPath?: string;
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
// shape the composer menu renders. Selecting it inserts a directive pointing
// the agent at the command file (see renderCommandDirective), so it never takes
// args in the composer.
function toSlashCommand(c: SlashUserCommand): SlashCommand {
  return {
    command: `/${c.name}`,
    description: c.description || c.path,
    takesArgs: false,
    isUserCommand: true,
    commandPath: c.path,
  };
}

// toSkillSlashCommand adapts an /api/commands skill into a SlashCommand entry.
// Selecting it inserts a "Use the skill …" directive (see renderSkillDirective),
// so it never takes args and carries no body.
function toSkillSlashCommand(s: SlashSkill): SlashCommand {
  return {
    command: `/${s.name}`,
    description: s.description,
    takesArgs: false,
    isSkill: true,
    skillPath: s.path,
  };
}

// fetchUserSlashCommands loads user-defined commands AND skills from
// /api/commands and maps them to SlashCommand entries, skipping any whose name
// collides with a hardcoded built-in. Commands come first, skills after.
// Callers merge these alongside SLASH_COMMANDS. Failures resolve to an empty
// list so the composer degrades to built-ins only.
export async function fetchUserSlashCommands(cwd?: string): Promise<SlashCommand[]> {
  try {
    const data = await api.getCommands(cwd);
    const commands = (data.user_commands ?? [])
      .filter((c) => !BUILTIN_NAMES.has(c.name.toLowerCase()))
      .map(toSlashCommand);
    const skills = (data.skills ?? [])
      .filter((s) => !BUILTIN_NAMES.has(s.name.toLowerCase()))
      .map(toSkillSlashCommand);
    return [...commands, ...skills];
  } catch {
    return [];
  }
}

// renderUseDirective builds the text inserted into the composer when a skill or
// command is selected. Rather than pasting the file's contents, it points the
// agent at the file by name and path so the agent reads/activates it. Built-in
// items have no filesystem path, so we omit the "located at" clause for them.
function renderUseDirective(kind: "skill" | "command", name: string, path?: string): string {
  return path ? `Use the ${kind} ${name} located at ${path}` : `Use the ${kind} ${name}`;
}

export function renderSkillDirective(name: string, path?: string): string {
  return renderUseDirective("skill", name, path);
}

export function renderCommandDirective(name: string, path?: string): string {
  return renderUseDirective("command", name, path);
}
