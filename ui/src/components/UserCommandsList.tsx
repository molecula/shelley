import React, { useEffect, useState } from "react";
import { api, SlashUserCommand } from "../services/api";
import CommandViewerModal from "./CommandViewerModal";

interface UserCommandsListProps {
  cwd?: string;
}

function UserCommandsList({ cwd }: UserCommandsListProps) {
  const [commands, setCommands] = useState<SlashUserCommand[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewing, setViewing] = useState<SlashUserCommand | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api
      .getSlashCommands(cwd)
      .then((d) => {
        if (!cancelled) {
          setCommands([...d.user_commands].sort((a, b) => a.name.localeCompare(b.name)));
          setError(null);
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err.message || String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [cwd]);

  if (loading) {
    return <div className="drawer-empty-state text-secondary"><p>Loading…</p></div>;
  }
  if (error) {
    return <div className="drawer-empty-state text-secondary"><p>Failed: {error}</p></div>;
  }
  if (commands.length === 0) {
    return (
      <div className="drawer-empty-state text-secondary">
        <p>No user-defined commands found.</p>
        <p className="skills-list-hint">
          Drop a markdown file into <code>~/.claude/commands/</code> or <code>.claude/commands/</code>.
        </p>
      </div>
    );
  }

  return (
    <>
      <ul className="skills-list">
        {commands.map((c) => (
          <li key={`${c.scope}:${c.name}`}>
            <button type="button" className="skills-list-item" onClick={() => setViewing(c)}>
              <span className="skills-list-name">/{c.name}</span>
              <span className="skills-list-desc">{c.description || c.path}</span>
              <span className={`skills-list-scope skills-list-scope-${c.scope}`}>{c.scope}</span>
            </button>
          </li>
        ))}
      </ul>
      {viewing && (
        <CommandViewerModal command={viewing} onClose={() => setViewing(null)} />
      )}
    </>
  );
}

export default UserCommandsList;