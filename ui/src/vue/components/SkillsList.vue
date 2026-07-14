<!-- Library > Skills tab. Lists all skills (user, project, builtin) from
     /api/user-skills; clicking one opens its SKILL.md in SkillViewerModal.
     Vue port of components/SkillsList.tsx. -->
<template>
  <div v-if="loading" class="drawer-empty-state text-secondary">
    <p>{{ t("loading") }}</p>
  </div>
  <div v-else-if="error" class="drawer-empty-state text-secondary">
    <p>{{ t("failedToLoad") }}: {{ error }}</p>
  </div>
  <div v-else-if="skills.length === 0" class="drawer-empty-state text-secondary">
    <p>{{ t("noUserSkills") }}</p>
    <p class="skills-list-hint">
      {{ t("skillsListHint") }}
    </p>
  </div>
  <template v-else>
    <ul class="skills-list">
      <li v-for="s in skills" :key="`${s.scope}:${s.name}`">
        <button type="button" class="skills-list-item" @click="viewing = s.name">
          <span class="skills-list-name">{{ s.name }}</span>
          <span class="skills-list-desc" :title="s.description">{{ truncate(s.description) }}</span>
          <span :class="`skills-list-scope skills-list-scope-${s.scope}`">{{ s.scope }}</span>
        </button>
      </li>
    </ul>
    <SkillViewerModal
      v-if="viewing"
      :name="viewing"
      :cwd="cwd"
      @close="viewing = null"
      @use="onUse"
    />
  </template>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api, type UserSkillSummary } from "../../services/api";
import { useI18n } from "../composables/i18n";
import SkillViewerModal from "./SkillViewerModal.vue";

const props = defineProps<{ cwd?: string }>();
const emit = defineEmits<{ (e: "use", text: string): void }>();

const { t } = useI18n();
const skills = ref<UserSkillSummary[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const viewing = ref<string | null>(null);

function onUse(text: string) {
  emit("use", text);
  viewing.value = null;
}

// Truncate long skill descriptions in the list so each row stays compact; the
// full text remains available via the element's title tooltip.
const DESC_MAX = 100;
function truncate(text: string): string {
  return text.length > DESC_MAX ? `${text.slice(0, DESC_MAX).trimEnd()}…` : text;
}

onMounted(async () => {
  try {
    const data = await api.getUserSkills(props.cwd);
    skills.value = [...data].sort((a, b) => a.name.localeCompare(b.name));
    error.value = null;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
});
</script>
