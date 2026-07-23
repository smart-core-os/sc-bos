<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">Points list</v-list-subheader>
      <v-list-item>
        <v-list-item-subtitle class="text-wrap">
          Download the distinct messages this automation has published since it last (re)started,
          as a CSV points list.
        </v-list-item-subtitle>
      </v-list-item>
      <v-card-actions class="justify-end px-4">
        <v-btn
            color="primary"
            variant="flat"
            prepend-icon="mdi-download"
            :loading="tracker.loading"
            :disabled="!automationName"
            @click="downloadPointsList">
          Download points list
        </v-btn>
      </v-card-actions>
    </v-list>
    <v-snackbar v-model="showEmpty" color="info" timeout="5000">
      No points found to export.
    </v-snackbar>
  </v-card>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import {listExportedPoints} from '@/api/ui/udmiExport';
import {useErrorStore} from '@/components/ui-error/error';
import {buildPointsCsv} from '@/routes/automations/pointsExport';
import {useSidebarStore} from '@/stores/sidebar';
import {dateStamp} from '@/util/date';
import {downloadCSVRows} from '@/util/downloadCSV';
import {computed, onMounted, onUnmounted, reactive, ref} from 'vue';

const sidebar = useSidebarStore();
const tracker = reactive(/** @type {ActionTracker<ListExportedPointsResponse.AsObject>} */ newActionTracker());
const showEmpty = ref(false);

const automationName = computed(() => sidebar.data?.config?.name ?? '');

const errorStore = useErrorStore();
let unwatchError;
onMounted(() => {
  unwatchError = errorStore.registerTracker(tracker);
});
onUnmounted(() => {
  unwatchError?.();
});

/**
 * Fetches the exported messages and downloads them as a CSV points list (one row per device).
 *
 * @return {Promise<void>}
 */
async function downloadPointsList() {
  if (!automationName.value) return;
  try {
    const res = await listExportedPoints({name: automationName.value}, tracker);
    const rows = buildPointsCsv(res?.messagesList ?? []);
    // rows always contains the header row; a length of 1 means nothing was exported.
    if (rows.length <= 1) {
      showEmpty.value = true;
      return;
    }
    downloadCSVRows(`points-list - ${automationName.value} - ${dateStamp()}.csv`, rows);
  } catch {
    // listExportedPoints failures are surfaced via the error store (tracker registered in
    // onMounted); swallow here so the @click handler doesn't raise an unhandled rejection.
  }
}
</script>

<style>
</style>
