<!-- Library > Commands tab. Lists user-defined slash commands from
     /api/commands (user_commands); clicking one opens its detail in
     CommandViewerModal. Vue port of components/UserCommandsList.tsx. -->
<template>
  <div v-if="loading" class="drawer-empty-state text-secondary">
    <p>{{ t("loading") }}</p>
  </div>
  <div v-else-if="error" class="drawer-empty-state text-secondary">
    <p>{{ t("failedToLoad") }}: {{ error }}</p>
  </div>
  <div v-else-if="commands.length === 0" class="drawer-empty-state text-secondary">
    <p>{{ t("noUserCommands") }}</p>
    <p class="skills-list-hint">
      {{ t("commandsListHint") }}
    </p>
  </div>
  <template v-else>
    <ul class="skills-list">
      <li v-for="c in commands" :key="`${c.scope}:${c.name}`">
        <button type="button" class="skills-list-item" @click="viewing = c">
          <span class="skills-list-name">/{{ c.name }}</span>
          <span class="skills-list-desc">{{ c.description || c.path }}</span>
          <span :class="`skills-list-scope skills-list-scope-${c.scope}`">{{ c.scope }}</span>
        </button>
      </li>
    </ul>
    <CommandViewerModal
      v-if="viewing"
      :command="viewing"
      @close="viewing = null"
      @use="onUse"
    />
  </template>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api, type SlashUserCommand } from "../../services/api";
import { useI18n } from "../composables/i18n";
import CommandViewerModal from "./CommandViewerModal.vue";

const props = defineProps<{ cwd?: string }>();
const emit = defineEmits<{ (e: "use", text: string): void }>();

const { t } = useI18n();
const commands = ref<SlashUserCommand[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const viewing = ref<SlashUserCommand | null>(null);

function onUse(text: string) {
  emit("use", text);
  viewing.value = null;
}

onMounted(async () => {
  try {
    const data = await api.getCommands(props.cwd);
    commands.value = [...data.user_commands].sort((a, b) => a.name.localeCompare(b.name));
    error.value = null;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
});
</script>
