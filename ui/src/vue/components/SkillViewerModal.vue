<!-- Read-only viewer for a Library skill. Fetches the full SKILL.md via
     /api/user-skills/{name} and renders it with MarkdownContent inside the
     shared Modal. Vue port of components/SkillViewerModal.tsx. -->
<template>
  <Modal :is-open="true" :title="name" class-name="skill-viewer-modal" @close="emit('close')">
    <template #title-right>
      <button type="button" class="btn btn-primary btn-sm" @click="onUse">
        {{ t("useThis") }}
      </button>
    </template>
    <p v-if="error" class="text-secondary">{{ t("failedToLoad") }}: {{ error }}</p>
    <p v-else-if="!data" class="text-secondary">{{ t("loading") }}</p>
    <template v-else>
      <p class="skill-viewer-path" :title="data.path">{{ data.path }}</p>
      <div class="skill-viewer-content">
        <MarkdownContent :text="data.content" />
      </div>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import Modal from "./Modal.vue";
import MarkdownContent from "./MarkdownContent.vue";
import { api, type UserSkillContent } from "../../services/api";
import { useI18n } from "../composables/i18n";

const props = defineProps<{ name: string; cwd?: string }>();
const emit = defineEmits<{ (e: "close"): void; (e: "use", text: string): void }>();

const { t } = useI18n();
const data = ref<UserSkillContent | null>(null);
const error = ref<string | null>(null);

// Insert a short "/name" directive token into the composer; the full directive
// text is substituted on submit (see MessageInput.expandDirectives).
function onUse() {
  emit("use", `/${props.name}`);
}

onMounted(async () => {
  try {
    data.value = await api.getUserSkillContent(props.name, props.cwd);
    error.value = null;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
});
</script>
