<!-- Read-only viewer for a Library user command. The command payload (body,
     description, argument-hint) already comes from /api/commands, so no extra
     fetch is needed. Vue port of components/CommandViewerModal.tsx. -->
<template>
  <Modal
    :is-open="true"
    :title="`/${command.name}`"
    class-name="skill-viewer-modal"
    @close="emit('close')"
  >
    <template #title-right>
      <button type="button" class="btn btn-primary btn-sm" @click="onUse">
        {{ t("useThis") }}
      </button>
    </template>
    <p class="skill-viewer-path" :title="command.path">{{ command.path }}</p>
    <p v-if="command.description" class="command-viewer-description">
      {{ command.description }}
    </p>
    <p v-if="command.argument_hint" class="command-viewer-hint">
      <strong>{{ t("argumentHint") }}:</strong> <code>{{ command.argument_hint }}</code>
    </p>
    <div class="skill-viewer-content">
      <MarkdownContent :text="command.body" />
    </div>
  </Modal>
</template>

<script setup lang="ts">
import Modal from "./Modal.vue";
import MarkdownContent from "./MarkdownContent.vue";
import type { SlashUserCommand } from "../../services/api";
import { useI18n } from "../composables/i18n";

const props = defineProps<{ command: SlashUserCommand }>();
const emit = defineEmits<{ (e: "close"): void; (e: "use", text: string): void }>();

const { t } = useI18n();

// Insert a short "/name" directive token into the composer; the full directive
// text is substituted on submit (see MessageInput.expandDirectives).
function onUse() {
  emit("use", `/${props.command.name}`);
}
</script>
