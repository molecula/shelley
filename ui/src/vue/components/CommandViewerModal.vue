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

defineProps<{ command: SlashUserCommand }>();
const emit = defineEmits<{ (e: "close"): void }>();

const { t } = useI18n();
</script>
