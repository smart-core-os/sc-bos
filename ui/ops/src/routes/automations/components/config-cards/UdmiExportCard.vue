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
      <v-list-item>
        <v-btn-toggle v-model="mode" density="compact" variant="outlined" divided mandatory>
          <v-btn value="device" size="small" text="Per device"/>
          <v-btn value="type" size="small" text="Per device type"/>
        </v-btn-toggle>
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
  </v-card>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import {listExportedPoints} from '@/api/ui/udmiExport';
import {useErrorStore} from '@/components/ui-error/error';
import {buildPointsCsv} from '@/routes/automations/pointsExport';
import {useSidebarStore} from '@/stores/sidebar';
import {downloadCSVRows} from '@/util/downloadCSV';
import {computed, onMounted, onUnmounted, reactive, ref} from 'vue';

const sidebar = useSidebarStore();
const tracker = reactive(/** @type {ActionTracker<ListExportedPointsResponse.AsObject>} */ newActionTracker());

const automationName = computed(() => sidebar.data?.config?.name ?? '');

// Export grouping mode: 'device' = one row per device; 'type' = one row per distinct
// point set within a device type (name prefix before the first hyphen).
const mode = ref('device');

const errorStore = useErrorStore();
let unwatchError;
onMounted(() => {
  unwatchError = errorStore.registerTracker(tracker);
});
onUnmounted(() => {
  unwatchError?.();
});

/**
 * Fetches the exported messages and downloads them as a CSV points list. The rows are
 * built either per device or per device type depending on the selected mode.
 *
 * @return {Promise<void>}
 */
async function downloadPointsList() {
  if (!automationName.value) return;
  const res = await listExportedPoints({name: automationName.value}, tracker);
  const rows = buildPointsCsv(res?.messagesList ?? [], mode.value);

  const dateString = new Date().toISOString().slice(0, 10);
  const kind = mode.value === 'type' ? 'points-list-by-type' : 'points-list';
  downloadCSVRows(`${kind} - ${automationName.value} - ${dateString}.csv`, rows);
}
</script>

<style>
</style>
