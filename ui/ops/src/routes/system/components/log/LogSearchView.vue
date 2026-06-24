<template>
  <div class="log-search-view">
    <v-text-field
        v-model="search"
        class="flex-grow-0 mb-2"
        clearable
        density="compact"
        hide-details
        placeholder="Search logs…"
        prepend-inner-icon="mdi-magnify"
        @keydown.enter.exact.prevent="nextMatch"
        @keydown.enter.shift.prevent="prevMatch">
      <template #append-inner>
        <span v-if="search" class="text-caption text-no-wrap text-neutral-lighten-4 mr-1">
          {{ matchCount ? activeMatch + 1 : 0 }}/{{ matchCount }}
        </span>
        <v-btn
            :disabled="!matchCount"
            density="compact"
            icon="mdi-chevron-up"
            size="small"
            title="Previous match (Shift+Enter)"
            variant="text"
            @click="prevMatch"/>
        <v-btn
            :disabled="!matchCount"
            density="compact"
            icon="mdi-chevron-down"
            size="small"
            title="Next match (Enter)"
            variant="text"
            @click="nextMatch"/>
      </template>
    </v-text-field>
    <log-viewport
        :messages="messages"
        :search="debouncedSearch"
        :active-match="activeMatch"
        class="flex-grow-1"
        @match-count="onMatchCount"/>
  </div>
</template>

<script setup>
import LogViewport from '@/routes/system/components/log/LogViewport.vue';
import debounce from 'debounce';
import {ref, watch} from 'vue';

defineProps({
  messages: {
    type: Array,
    default: () => []
  }
});

// Ctrl+F style search: highlights matches in the viewport without filtering.
// The viewport re-splits lines when the term changes, so debounce keystrokes.
const search = ref('');
const debouncedSearch = ref('');
const applySearch = debounce(v => {
  debouncedSearch.value = v || '';
}, 200);
const matchCount = ref(0);
const activeMatch = ref(0);

watch(search, applySearch);
watch(debouncedSearch, () => {
  activeMatch.value = 0;
});

/**
 * @param {number} count
 */
function onMatchCount(count) {
  matchCount.value = count;
  if (activeMatch.value >= count) activeMatch.value = Math.max(0, count - 1);
}

/**
 * Moves the active highlight to the next match, wrapping around.
 */
function nextMatch() {
  if (matchCount.value) activeMatch.value = (activeMatch.value + 1) % matchCount.value;
}

/**
 * Moves the active highlight to the previous match, wrapping around.
 */
function prevMatch() {
  if (matchCount.value) activeMatch.value = (activeMatch.value - 1 + matchCount.value) % matchCount.value;
}
</script>

<style scoped>
.log-search-view {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* allow the viewport to shrink within a height-capped container */
.log-search-view > :deep(.log-viewport) {
  min-height: 0;
}
</style>
