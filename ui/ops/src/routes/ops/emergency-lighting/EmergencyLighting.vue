<template>
  <content-card>
    <v-card-title class="d-flex">
      <h4 class="text-h4">Emergency Lighting</h4>
      <v-spacer/>
      <v-btn
          color="primary"
          :disabled="blockActions"
          @click="downloadCSV">
        Download CSV
        <v-icon end>mdi-download</v-icon>
      </v-btn>
    </v-card-title>
    <v-data-table
        :headers="headers"
        :items="lightHealth"
        :loading="lightHealthTracker.loading"
        show-select
        item-value="name"
        v-model="selectedLights"
        item-key="name">
      <template #top>
        <span v-if="selectedLights.length > 0">
          <v-btn
              color="accent-darken-1"
              class="ml-4"
              :disabled="blockActions"
              @click="functionTest">Function Test</v-btn>
          <v-btn
              color="accent-darken-1"
              class="ml-4"
              :disabled="blockActions"
              @click="durationTest">Duration Test</v-btn>
          <span class="pl-4">
            {{ selectedLights.length }} light{{ selectedLights.length === 1 ? '' : 's' }} selected
          </span>
        </span>
      </template>
      <template #item.faultsList="{ value }">
        <span class="text-title-bold text-success-lighten-3" v-if="value.length === 0">OK</span>
        <span class="text-title-bold text-error-lighten-1" v-else>
          {{ value.map((v) => faultToString(v)).join(', ') }}
        </span>
      </template>
      <template #item.updateTime="{ item }">
        <span v-if="timestampToDate(item.updateTime)">
          {{ timestampToDate(item.updateTime).toLocaleString() }}
        </span>
      </template>
      <template #item.lastFunctionTest="{ item }">
        <span v-if="timestampToDate(item.lastFunctionTest)">
          {{ timestampToDate(item.lastFunctionTest).toLocaleString() }}
        </span>
      </template>
      <template #item.lastDurationTest="{ item }">
        <span v-if="timestampToDate(item.lastDurationTest)">
          {{ timestampToDate(item.lastDurationTest).toLocaleString() }}
        </span>
      </template>
    </v-data-table>
  </content-card>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import {newActionTracker} from '@/api/resource';
import {faultToString, getReportCSV, listLightHealth, runTest} from '@/api/sc/traits/lighting-test';
import ContentCard from '@/components/ContentCard.vue';
import {useErrorStore} from '@/components/ui-error/error';
import useAuthSetup from '@/composables/useAuthSetup';
import {EmergencyStatus} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/driver/dali/v1/dali_pb';
import {computed, onMounted, onUnmounted, reactive, ref} from 'vue';

const {blockActions} = useAuthSetup();

const headers = [
  {title: 'Name', key: 'name'},
  {title: 'Status', key: 'faultsList'},
  {title: 'Updated', key: 'updateTime'},
  {title: 'Last Function Test', key: 'lastFunctionTest'},
  {title: 'Last Duration Test', key: 'lastDurationTest'}
];

const selectedLights = ref([]);

const csvTracker = reactive(/** @type {ActionTracker<ReportCSV.AsObject>} */ newActionTracker());
const lightHealthTracker = reactive(/** @type {ActionTracker<ListLightHealthResponse.AsObject>}*/ newActionTracker());
const allLightsHealth = ref([]);

const lightHealth = computed(() => {
  return allLightsHealth.value;
});

onMounted(() => refreshLightHealth().catch((err) => console.error('Error fetching light health: ', err)));
onUnmounted(() => (allLightsHealth.value = []));

// Ui Error handling
const errorStore = useErrorStore();
let unwatchCsvErrors;
let unwatchLightHealthErrors;
onMounted(() => {
  unwatchCsvErrors = errorStore.registerTracker(csvTracker);
  unwatchLightHealthErrors = errorStore.registerTracker(lightHealthTracker);
});
onUnmounted(() => {
  if (unwatchCsvErrors) unwatchCsvErrors();
  if (unwatchLightHealthErrors) unwatchLightHealthErrors();
});

/**
 *
 */
async function refreshLightHealth() {
  const req = {pageSize: 100};
  while (true) {
    const resp = await listLightHealth(req, lightHealthTracker);
    allLightsHealth.value.push(...resp.emergencyLightsList);
    if (!resp.nextPageToken) break;
    req.pageToken = resp.nextPageToken;
  }
}

/**
 * download the CSV report, fetched from the server
 */
async function downloadCSV() {
  const csvObj = await getReportCSV(csvTracker);
  // base64 decode contents
  const csvContent = atob(csvObj.csv);
  // create fake file to serve
  const anchor = document.createElement('a');
  anchor.href = 'data:text/csv;charset=utf-8,' + encodeURIComponent(csvContent);
  anchor.target = '_blank';
  anchor.download = 'emergency-lighting-report.csv';
  anchor.click();
}

/**
 * @return {Promise<void>}
 */
function durationTest() {
  return doTest(EmergencyStatus.Test.DURATION_TEST);
}

/**
 * @return {Promise<void>}
 */
function functionTest() {
  return doTest(EmergencyStatus.Test.FUNCTION_TEST);
}

/**
 *
 * @param {EmergencyStatus.Test} type
 */
async function doTest(type) {
  const lightingTests = selectedLights.value.map((light) => {
    const req = {
      name: light,
      test: type
    };
    return runTest(req);
  });
  await Promise.all(lightingTests).catch((err) => console.error('Error running test: ', err));
  selectedLights.value = [];
}
</script>

<style scoped></style>