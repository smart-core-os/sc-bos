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
  const messages = res?.messagesList ?? [];

  const {header, rows} = mode.value === 'type' ? rowsByDeviceType(messages) : rowsByDevice(messages);

  const dateString = new Date().toISOString().slice(0, 10);
  const kind = mode.value === 'type' ? 'points-list-by-type' : 'points-list';
  downloadCSVRows(`${kind} - ${automationName.value} - ${dateString}.csv`, [header, ...rows]);
}

/**
 * Builds one row per device — `Source name, Topic, Point 1..N` — the row widening to hold
 * each device's point names.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {{header: string[], rows: string[][]}}
 */
function rowsByDevice(messages) {
  let maxPoints = 0;
  const rows = messages.map((msg) => {
    const points = parsePoints(msg.payload);
    maxPoints = Math.max(maxPoints, points.length);
    return [msg.sourceName, msg.topic, ...points];
  });
  return {header: ['Source name', 'Topic', ...pointColumns(maxPoints)], rows};
}

/**
 * Builds one row per distinct point set within a device type — `Device type, Devices,
 * Point 1..N`. Devices of a type are grouped, but only those with an identical point set
 * are collapsed, so a subset/superset variant appears as its own (longer) row rather than
 * being silently merged. Types are sorted alphabetically; variants within a type by size.
 *
 * @param {Array<{sourceName: string, topic: string, payload: string}>} messages
 * @return {{header: string[], rows: string[][]}}
 */
function rowsByDeviceType(messages) {
  // type -> (signature -> {points, count})
  const byType = new Map();
  for (const msg of messages) {
    const type = deviceType(msg.sourceName);
    const points = parsePoints(msg.payload);
    const signature = JSON.stringify([...points].sort());
    if (!byType.has(type)) byType.set(type, new Map());
    const variants = byType.get(type);
    const existing = variants.get(signature);
    if (existing) existing.count++;
    else variants.set(signature, {points, count: 1});
  }

  let maxPoints = 0;
  const rows = [];
  for (const type of [...byType.keys()].sort()) {
    const variants = [...byType.get(type).values()].sort((a, b) => a.points.length - b.points.length);
    for (const {points, count} of variants) {
      maxPoints = Math.max(maxPoints, points.length);
      rows.push([type, String(count), ...points]);
    }
  }
  return {header: ['Device type', 'Devices', ...pointColumns(maxPoints)], rows};
}

/**
 * Returns `n` CSV column headers named "Point 1" .. "Point n".
 *
 * @param {number} n
 * @return {string[]}
 */
function pointColumns(n) {
  return Array.from({length: n}, (_, i) => `Point ${i + 1}`);
}

/**
 * Derives a device type from a Smart Core source name: the token before the first hyphen
 * of the last '/'-segment (e.g. "a/b/FCU-LN1-01" -> "FCU"). Falls back to the whole short
 * name when there is no hyphen.
 *
 * @param {string} sourceName
 * @return {string}
 */
function deviceType(sourceName) {
  return (sourceName.split('/').pop() ?? '').split('-')[0];
}

/**
 * Extracts the point names from a UDMI pointset payload. Handles both the conformant
 * envelope (`{points: {name: {...}}}`) and a bare points map (`{name: {present_value}}`,
 * as the mock driver emits). Returns an empty array for payloads that aren't a pointset
 * (state/metadata/other).
 *
 * @param {string} payload
 * @return {string[]}
 */
function parsePoints(payload) {
  let parsed;
  try {
    parsed = JSON.parse(payload);
  } catch {
    return [];
  }
  if (!parsed || typeof parsed !== 'object') return [];
  // Prefer the conformant envelope; fall back to treating the whole object as the
  // points map when its values look like point values (have a present_value).
  let points = parsed.points;
  if (!points || typeof points !== 'object') {
    const looksLikePointsMap = Object.values(parsed).some(
        (v) => v && typeof v === 'object' && Object.hasOwn(v, 'present_value'));
    if (!looksLikePointsMap) return [];
    points = parsed;
  }
  return Object.keys(points);
}
</script>

<style>
</style>
