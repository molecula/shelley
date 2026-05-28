import React, { useState, useEffect, useCallback } from "react";
import Modal from "./Modal";
import { useI18n } from "../i18n";
import { scheduledTasksApi, ScheduledTask } from "../services/api";

interface ScheduledTasksModalProps {
  isOpen: boolean;
  onClose: () => void;
}

// Convert systemd microsecond timestamps ("0" / "n/a" => empty) to locale strings.
function formatUSec(usec: string): string {
  if (!usec || usec === "0" || usec === "n/a") return "—";
  const n = Number(usec);
  if (!Number.isFinite(n) || n <= 0) return usec;
  return new Date(n / 1000).toLocaleString();
}

function ScheduledTasksModal({ isOpen, onClose }: ScheduledTasksModalProps) {
  const { t } = useI18n();
  const [tasks, setTasks] = useState<ScheduledTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      setTasks(await scheduledTasksApi.list());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) load();
  }, [isOpen, load]);

  const handleDelete = async (name: string) => {
    try {
      setPendingDelete(name);
      await scheduledTasksApi.remove(name);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPendingDelete(null);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t("scheduledTasks")} className="scheduled-tasks-modal-wrapper">
      <div className="scheduled-tasks-modal">
        {loading && <div className="scheduled-tasks-empty">{t("loading")}…</div>}
        {error && <div className="scheduled-tasks-error">{error}</div>}
        {!loading && !error && tasks.length === 0 && (
          <div className="scheduled-tasks-empty">{t("noScheduledTasks")}</div>
        )}
        {!loading && tasks.length > 0 && (
          <table className="scheduled-tasks-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>{t("schedule")}</th>
                <th>{t("nextFire")}</th>
                <th>{t("lastFire")}</th>
                <th>{t("command")}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.name}>
                  <td>
                    <div className="scheduled-tasks-name">{task.name}</div>
                    {task.description && (
                      <div className="scheduled-tasks-desc">{task.description}</div>
                    )}
                  </td>
                  <td>
                    <code>{task.on_calendar || "—"}</code>
                    {task.persistent && (
                      <div className="scheduled-tasks-flag">{t("persistent")}</div>
                    )}
                  </td>
                  <td>{formatUSec(task.next_fire)}</td>
                  <td>{formatUSec(task.last_fire)}</td>
                  <td>
                    <code className="scheduled-tasks-exec">{task.exec_start || "—"}</code>
                  </td>
                  <td>
                    <button
                      className="scheduled-tasks-delete"
                      onClick={() => handleDelete(task.name)}
                      disabled={pendingDelete === task.name}
                      title={t("removeScheduledTask")}
                    >
                      {pendingDelete === task.name ? "…" : "✕"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </Modal>
  );
}

export default ScheduledTasksModal;
