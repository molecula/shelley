<!-- Full detail popup for a single scheduled task, opened by clicking a card in
     ScheduledTasksView. Shows the schedule, run times, working directory and the
     full prompt, and lets the user delete the task. (A per-task run history is
     added below the prompt once a run data source is wired up.) -->
<template>
  <Modal :is-open="true" :title="title" class-name="scheduled-task-modal" @close="emit('close')">
    <template #title-right>
      <span :class="['scheduled-task-badge', task.status]">
        {{ statusLabel(task.status) }}
      </span>
    </template>

    <dl class="scheduled-task-detail-meta">
      <div class="scheduled-task-field">
        <dt>{{ t("taskSchedule") }}</dt>
        <dd>
          {{ task.scheduleLabel || task.schedule || "—" }}
          <code v-if="task.scheduleLabel && task.schedule" class="scheduled-task-schedule-raw">{{
            task.schedule
          }}</code>
        </dd>
      </div>
      <div v-if="task.nextRun" class="scheduled-task-field">
        <dt>{{ t("taskNextRun") }}</dt>
        <dd>{{ task.nextRun }}</dd>
      </div>
      <div v-if="task.lastRun" class="scheduled-task-field">
        <dt>{{ t("taskLastRun") }}</dt>
        <dd>{{ task.lastRun }}</dd>
      </div>
      <div v-if="task.cwd" class="scheduled-task-field">
        <dt>{{ t("taskCwd") }}</dt>
        <dd>
          <code>{{ tildifyPath(task.cwd) }}</code>
        </dd>
      </div>
    </dl>

    <div v-if="task.prompt" class="scheduled-task-detail-prompt">
      <span class="scheduled-task-label">{{ t("taskPrompt") }}</span>
      <p>{{ task.prompt }}</p>
    </div>

    <div class="scheduled-task-runs">
      <span class="scheduled-task-label">{{ t("taskRuns") }}</span>
      <p v-if="runsLoading" class="scheduled-task-runs-empty text-secondary">{{ t("loading") }}</p>
      <p v-else-if="runsError" class="scheduled-task-runs-empty text-secondary">{{ runsError }}</p>
      <p v-else-if="runs.length === 0" class="scheduled-task-runs-empty text-secondary">
        {{ t("noTaskRuns") }}
      </p>
      <ul v-else class="scheduled-task-runs-list">
        <li v-for="(run, i) in runs" :key="run.conversationId + i">
          <button
            type="button"
            class="scheduled-task-run"
            :disabled="!run.conversationId"
            :title="run.conversationId ? t('openRunConversation') : ''"
            @click="openRun(run)"
          >
            <span class="scheduled-task-run-time">{{ formatTs(run.ts) }}</span>
            <i v-if="run.conversationId" class="pi pi-arrow-up-right" aria-hidden="true" />
          </button>
        </li>
      </ul>
    </div>

    <p v-if="error" class="text-secondary scheduled-task-error">{{ error }}</p>

    <div class="scheduled-task-detail-footer">
      <button class="btn btn-danger btn-sm" :disabled="deleting" @click="onDelete">
        <i class="pi pi-trash" aria-hidden="true" />
        {{ t("deleteTask") }}
      </button>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Modal from "./Modal.vue";
import {
  api,
  type ScheduledTask,
  type ScheduledRun,
  type ScheduledTaskStatus,
} from "../../services/api";
import { useI18n } from "../composables/i18n";
import { tildifyPath } from "../../utils/tildify";

const props = defineProps<{ task: ScheduledTask }>();
const emit = defineEmits<{
  (e: "close"): void;
  (e: "deleted", name: string): void;
  (e: "open-conversation", conversationId: string, slug: string): void;
}>();

const { t } = useI18n();
const deleting = ref(false);
const error = ref<string | null>(null);
const runs = ref<ScheduledRun[]>([]);
const runsLoading = ref(true);
const runsError = ref<string | null>(null);

const title = computed(() =>
  props.task.name.startsWith("shelley-")
    ? props.task.name.slice("shelley-".length)
    : props.task.name,
);

function statusLabel(status: ScheduledTaskStatus): string {
  if (status === "active") return t("taskActive");
  if (status === "completed") return t("taskCompleted");
  return t("taskInactive");
}

// Render the stored RFC3339 timestamp in the viewer's locale; fall back to the
// raw string if it doesn't parse.
function formatTs(ts: string): string {
  const d = new Date(ts);
  return isNaN(d.getTime()) ? ts : d.toLocaleString();
}

function openRun(run: ScheduledRun) {
  if (!run.conversationId) return;
  emit("open-conversation", run.conversationId, run.slug);
}

onMounted(async () => {
  try {
    runs.value = await api.getScheduledTaskRuns(props.task.name);
    runsError.value = null;
  } catch (err) {
    runsError.value = err instanceof Error ? err.message : String(err);
  } finally {
    runsLoading.value = false;
  }
});

async function onDelete() {
  if (!window.confirm(`${t("confirmDeleteTask")} (${title.value})`)) return;
  deleting.value = true;
  error.value = null;
  try {
    await api.deleteScheduledTask(props.task.name);
    emit("deleted", props.task.name);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    deleting.value = false;
  }
}
</script>
