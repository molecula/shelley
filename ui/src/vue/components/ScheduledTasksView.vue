<!-- Main-content view listing the recurring tasks created by the /schedule
     skill (systemd user timers). Takes over from ChatInterface when the
     "Scheduled Tasks" nav item is selected. Tasks are shown as a grid of
     compact cards; clicking a card opens the full detail in a modal. -->
<template>
  <div class="full-height flex flex-col scheduled-tasks-view">
    <!-- Header (mirrors ChatInterface's .header) -->
    <div class="header">
      <div class="header-left">
        <button
          class="btn-icon hide-on-desktop"
          :aria-label="t('openConversations')"
          @click="props.onOpenDrawer()"
        >
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        </button>
        <button
          v-if="isDrawerCollapsed && onToggleDrawerCollapse"
          class="btn-icon show-on-desktop-only"
          :aria-label="t('expandSidebar')"
          :title="t('expandSidebar')"
          @click="onToggleDrawerCollapse && onToggleDrawerCollapse()"
        >
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M13 5l7 7-7 7M5 5l7 7-7 7"
            />
          </svg>
        </button>
        <h1 class="header-title">{{ t("scheduledTasks") }}</h1>
      </div>
    </div>

    <!-- Body -->
    <div class="scheduled-tasks-body">
      <p v-if="platformSupported" class="scheduled-tasks-subtitle">
        <template v-if="hintParts.length === 2"
          >{{ hintParts[0] }}<code>/schedule</code>{{ hintParts[1] }}</template
        >
        <template v-else>{{ t("createTaskHint") }}</template>
      </p>

      <div v-if="loading" class="drawer-empty-state text-secondary">
        <p>{{ t("loading") }}</p>
      </div>
      <div v-else-if="error" class="drawer-empty-state text-secondary">
        <p>{{ t("failedToLoad") }}: {{ error }}</p>
        <button class="btn-secondary" @click="load">{{ t("retry") }}</button>
      </div>
      <div v-else-if="!platformSupported" class="drawer-empty-state text-secondary">
        <p>{{ t("schedulingUnavailable") }}</p>
        <p class="scheduled-tasks-hint">{{ t("schedulingUnavailableHint") }}</p>
      </div>
      <div v-else-if="tasks.length === 0" class="drawer-empty-state text-secondary">
        <p>{{ t("noScheduledTasks") }}</p>
        <p class="scheduled-tasks-hint">{{ t("noScheduledTasksHint") }}</p>
      </div>
      <div v-else class="scheduled-tasks-grid">
        <button
          v-for="task in tasks"
          :key="task.name"
          type="button"
          class="scheduled-task-card"
          @click="selected = task"
        >
          <div class="scheduled-task-head">
            <span class="scheduled-task-name">{{ displayName(task.name) }}</span>
            <span :class="['scheduled-task-badge', task.status]">
              {{ statusLabel(task.status) }}
            </span>
          </div>
          <div class="scheduled-task-card-meta">
            <span class="scheduled-task-schedule" :title="task.schedule"
              ><i class="pi pi-clock" aria-hidden="true" />
              {{ task.scheduleLabel || task.schedule || "—" }}</span
            >
            <span v-if="task.nextRun" class="scheduled-task-next">{{
              t("taskNextRun") + ": " + task.nextRun
            }}</span>
          </div>
        </button>
      </div>
    </div>

    <ScheduledTaskDetailModal
      v-if="selected"
      :task="selected"
      @close="selected = null"
      @deleted="onDeleted"
      @open-conversation="(id, slug) => emit('open-conversation', id, slug)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api, type ScheduledTask, type ScheduledTaskStatus } from "../../services/api";
import { useI18n } from "../composables/i18n";
import ScheduledTaskDetailModal from "./ScheduledTaskDetailModal.vue";

const props = defineProps<{
  onOpenDrawer: () => void;
  isDrawerCollapsed?: boolean;
  onToggleDrawerCollapse?: () => void;
}>();
const emit = defineEmits<{
  (e: "open-conversation", conversationId: string, slug: string): void;
}>();

const { t } = useI18n();
const tasks = ref<ScheduledTask[]>([]);
const platformSupported = ref(true);
const loading = ref(true);
const error = ref<string | null>(null);
const selected = ref<ScheduledTask | null>(null);

// Split the hint sentence around the literal "/schedule" token so it can be
// rendered as inline <code>. Falls back to the whole string if a translation
// omits the token.
const hintParts = computed(() => t("createTaskHint").split("/schedule"));

// Strip the "shelley-" prefix the units always carry for a friendlier label.
function displayName(name: string): string {
  return name.startsWith("shelley-") ? name.slice("shelley-".length) : name;
}

function statusLabel(status: ScheduledTaskStatus): string {
  if (status === "active") return t("taskActive");
  if (status === "completed") return t("taskCompleted");
  return t("taskInactive");
}

async function load() {
  loading.value = true;
  try {
    const data = await api.getScheduledTasks();
    platformSupported.value = data.platformSupported;
    tasks.value = data.tasks;
    error.value = null;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

function onDeleted(name: string) {
  tasks.value = tasks.value.filter((tk) => tk.name !== name);
  selected.value = null;
}

onMounted(load);
</script>
