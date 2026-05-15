import React, { useEffect, useMemo, useState } from "react";
import { api, SlashCommandsResponse } from "../services/api";

export type SlashItem =
  | { kind: "builtin"; name: string; description: string; action: string }
  | { kind: "user-command"; name: string; description: string; body: string; argumentHint?: string; scope: string }
  | { kind: "skill"; name: string; description: string };

interface SlashCommandPaletteProps {
  /** Open when query is non-null ("" means just typed `/`). */
  query: string | null;
  cwd?: string;
  onSelect: (item: SlashItem) => void;
  onClose: () => void;
  /** Bound to the input — caller forwards Arrow/Enter/Tab/Escape via setKeyHandler. */
  registerKeyHandler: (handler: ((e: KeyboardEvent) => boolean) | null) => void;
}

interface ScoredItem {
  item: SlashItem;
  score: number;
}

function fuzzyScore(query: string, text: string): number {
  if (!query) return 1;
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  if (t.startsWith(q)) return 1000 + (q.length / t.length) * 100;
  const idx = t.indexOf(q);
  if (idx >= 0) return 500 - idx;
  // subsequence match
  let qi = 0;
  for (let i = 0; i < t.length && qi < q.length; i++) {
    if (t[i] === q[qi]) qi++;
  }
  if (qi === q.length) return 100 - (t.length - q.length);
  return -1;
}

function SlashCommandPalette({
  query,
  cwd,
  onSelect,
  onClose,
  registerKeyHandler,
}: SlashCommandPaletteProps) {
  const [data, setData] = useState<SlashCommandsResponse | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Fetch on open / cwd change.
  useEffect(() => {
    if (query === null) return;
    let cancelled = false;
    api
      .getSlashCommands(cwd)
      .then((d) => {
        if (!cancelled) {
          setData(d);
          setLoadError(null);
        }
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err.message || String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [query === null, cwd]);

  const items: SlashItem[] = useMemo(() => {
    if (!data) return [];
    const out: SlashItem[] = [];
    for (const b of data.builtins)
      out.push({ kind: "builtin", name: b.name, description: b.description, action: b.action });
    for (const c of data.user_commands)
      out.push({
        kind: "user-command",
        name: c.name,
        description: c.description,
        body: c.body,
        argumentHint: c.argument_hint,
        scope: c.scope,
      });
    for (const s of data.skills)
      out.push({ kind: "skill", name: s.name, description: s.description });
    return out;
  }, [data]);

  const filtered: ScoredItem[] = useMemo(() => {
    if (query === null) return [];
    const q = query;
    const scored: ScoredItem[] = [];
    for (const item of items) {
      const nameScore = fuzzyScore(q, item.name);
      const descScore = fuzzyScore(q, item.description) * 0.3;
      const score = Math.max(nameScore, descScore);
      if (score > 0) scored.push({ item, score });
    }
    scored.sort((a, b) => b.score - a.score);
    return scored.slice(0, 50);
  }, [items, query]);

  // Reset selection on filter change.
  useEffect(() => {
    setSelectedIndex(0);
  }, [query, filtered.length]);

  // Wire keyboard handler. Returns true if the event is consumed.
  useEffect(() => {
    if (query === null) {
      registerKeyHandler(null);
      return;
    }
    const handler = (e: KeyboardEvent): boolean => {
      if (filtered.length === 0 && e.key !== "Escape") return false;
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1));
        return true;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
        return true;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        if (filtered.length === 0) return false;
        e.preventDefault();
        const picked = filtered[Math.min(selectedIndex, filtered.length - 1)].item;
        onSelect(picked);
        return true;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return true;
      }
      return false;
    };
    registerKeyHandler(handler);
    return () => registerKeyHandler(null);
  }, [query, filtered, selectedIndex, onSelect, onClose, registerKeyHandler]);

  if (query === null) return null;

  return (
    <div className="slash-palette" role="listbox" aria-label="Slash commands">
      {loadError && <div className="slash-palette-error">Failed to load: {loadError}</div>}
      {!loadError && filtered.length === 0 && (
        <div className="slash-palette-empty">No matching commands</div>
      )}
      {filtered.map(({ item }, i) => (
        <button
          key={`${item.kind}:${item.name}`}
          type="button"
          className={`slash-palette-item${i === selectedIndex ? " selected" : ""}`}
          onMouseDown={(e) => {
            // Prevent textarea blur before click handler runs.
            e.preventDefault();
            onSelect(item);
          }}
          onMouseEnter={() => setSelectedIndex(i)}
          role="option"
          aria-selected={i === selectedIndex}
        >
          <span className="slash-palette-name">
            /{item.name}
            {item.kind === "user-command" && item.argumentHint && (
              <span className="slash-palette-hint"> {item.argumentHint}</span>
            )}
          </span>
          <span className="slash-palette-desc">{item.description}</span>
          <span className={`slash-palette-badge slash-palette-badge-${item.kind}`}>
            {item.kind === "builtin" && "command"}
            {item.kind === "user-command" && (item.scope === "project" ? "project" : "user")}
            {item.kind === "skill" && "skill"}
          </span>
        </button>
      ))}
    </div>
  );
}

export default SlashCommandPalette;

// Render a user-command body, substituting $ARGUMENTS.
export function renderUserCommand(body: string, args: string): string {
  if (body.includes("$ARGUMENTS")) {
    return body.replace(/\$ARGUMENTS/g, args);
  }
  return args ? `${body}\n\n${args}` : body;
}