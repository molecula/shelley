<!-- HostIcon — fetches the LLM-generated SVG for this host from
     /api/host-icon and renders it inline so instances are visually
     distinguishable. Renders nothing when the server has no icon (404).
     v-html is acceptable here: the SVG is our own server-generated,
     trusted content, not user input. -->
<template>
  <div
    v-if="svgMarkup"
    class="host-icon"
    :style="{ width: size + 'px', height: size + 'px' }"
    :title="hostname"
    v-html="svgMarkup"
  />
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "../../services/api";

const props = withDefaults(
  defineProps<{
    hostname?: string;
    size?: number;
  }>(),
  { size: 22 },
);

const svgMarkup = ref<string | null>(null);

onMounted(async () => {
  try {
    const svg = await api.getHostIcon();
    if (svg) svgMarkup.value = svg;
  } catch {
    // No icon available — render nothing.
  }
});
</script>
