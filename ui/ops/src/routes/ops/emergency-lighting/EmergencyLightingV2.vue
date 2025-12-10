<template>
  <content-card>
    <v-card-title class="d-flex">
      <h4 class="text-h4">Emergency Lighting</h4>
      <v-spacer/>
      <v-progress-circular
          :value="totalDevices > 0 ? (loadedResults / totalDevices) * 100 : 0"
          color="primary"
          class="mt-2"
          striped
          :active="loadedResults < totalDevices">
        <template #default>
          <span v-if="totalDevices > 0" class="caption">
            {{ loadedResults }} / {{ totalDevices }} loaded
          </span>
        </template>
      </v-progress-circular>
      <v-spacer/>
      <v-btn
          color="primary"
          class="ml-2"
          :disabled="loadedResults < totalDevices"
          @click="refreshTable">
        Refresh
        <v-icon end>mdi-refresh</v-icon>
      </v-btn>
      <v-spacer/>
      <div class="d-flex align-center">
        <template v-if="hasFilters">
          <filter-choice-chips :ctx="filterCtx" class="mx-2"/>
          <filter-btn :ctx="filterCtx" flat/>
        </template>
        <v-tooltip text="Download data as CSV..." location="bottom">
          <template #activator="{ props }">
            <v-btn
                v-bind="props"
                flat
                class="ml-2"
                :disabled="loadedResults < totalDevices"
                @click="downloadCSV">
              <v-icon size="24" end>mdi-file-download</v-icon>
            </v-btn>
          </template>
        </v-tooltip>
      </div>
    </v-card-title>
    <v-data-table
        :headers="headers"
        :items="filteredTestResults"
        :items-per-page="50"
        show-select
        item-value="name"
        v-model="selectedLights"
        item-key="name">
      <template #top>
        <span v-if="selectedLights.length > 0">
          <v-btn
              color="primary"
              class="ml-4"
              @click="functionTest">Function Test</v-btn>
          <v-btn
              color="primary"
              class="ml-4"
              @click="durationTest">Duration Test</v-btn>
          <span class="pl-4">
            {{ selectedLights.length }} light{{ selectedLights.length === 1 ? '' : 's' }} selected
          </span>
        </span>
      </template>
      <template #item.functionTest.endTime="{ item }">
        <span v-if="item.functionTest && item.functionTest.endTime">
          {{ timestampToDate(item.functionTest.endTime).toLocaleString() }}
        </span>
      </template>
      <template #item.functionTest.result="{ item }">
        <span v-if="item.functionTest">
          {{ emergencyLightResultToString(item.functionTest.result) }}
        </span>
      </template>
      <template #item.durationTest.endTime="{ item }">
        <span v-if="item.durationTest && item.durationTest.endTime">
          {{ timestampToDate(item.durationTest.endTime).toLocaleString() }}
        </span>
      </template>
      <template #item.durationTest.result="{ item }">
        <span v-if="item.durationTest">
          {{ emergencyLightResultToString(item.durationTest.result) }}
        </span>
      </template>
    </v-data-table>
  </content-card>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import {
  emergencyLightResultToString, getTestResultSet,
  startDurationTest,
  startFunctionTest
} from '@/api/sc/traits/emergency-light.js';
import {listDevices} from '@/api/ui/devices.js';
import ContentCard from '@/components/ContentCard.vue';
import FilterBtn from '@/components/filter/FilterBtn.vue';
import FilterChoiceChips from '@/components/filter/FilterChoiceChips.vue';
import {useDeviceFilters} from '@/composables/devices';
import {ref, onMounted, computed, watch} from 'vue';

const forcedFilters = ref({
  'metadata.traits.name': 'smartcore.bos.EmergencyLight'
});
const {filterCtx, forcedConditions, filterConditions} = useDeviceFilters(forcedFilters);
const hasFilters = computed(() => filterCtx.filters.value.length > 0);

const headers = [
  {title: 'Name', key: 'name'},
  {title: 'Last Function Test', key: 'functionTest.endTime'},
  {title: 'Function Test Result', key: 'functionTest.result'},
  {title: 'Last Duration Test', key: 'durationTest.endTime'},
  {title: 'Duration Test Result', key: 'durationTest.result'}
];

const selectedLights = ref([]);
const testResults = ref([]);
const totalDevices = ref(0);
const loadedResults = ref(0);
let currentFetchId = 0; // Track the current fetch to ignore stale results

const findEmLightsQuery = computed(() => {
  const q = {conditionsList: []};
  q.conditionsList.push(...forcedConditions.value);
  q.conditionsList.push(...filterConditions.value);
  return q;
});

const filteredTestResults = computed(() => {
  return testResults.value;
});

const getDeviceTestResults = async () => {

  const fetchId = ++currentFetchId;

  testResults.value = [];
  loadedResults.value = 0;
  let pageToken = '';
  let allDevices = [];
  do {
    const collection = await listDevices({
      query: findEmLightsQuery.value,
      pageSize: 100,
      pageToken
    });
    pageToken = collection.nextPageToken;
    allDevices = allDevices.concat(collection.devicesList);
  } while (pageToken !== '');

  if (fetchId !== currentFetchId) return;

  totalDevices.value = allDevices.length;

  const resultsMap = new Map();

  for (const item of allDevices) {
    getTestResultSet({ name: item.name, queryDevice: true })
        .then(testResult => {

          if (fetchId !== currentFetchId) return;

          const result = {
            name: item.name,
            functionTest: testResult.functionTest,
            durationTest: testResult.durationTest
          };
          resultsMap.set(item.name, result);
          testResults.value = Array.from(resultsMap.values());
          loadedResults.value++;
        })
        .catch(err => {
          if (fetchId !== currentFetchId) return;

          console.error('Error fetching test results for device: ', item.name, err);
          const result = {
            name: item.name,
            functionTest: {
              testResult: -1,
            },
            durationTest: {
              testResult: -1,
            }
          };
          resultsMap.set(item.name, result);
          testResults.value = Array.from(resultsMap.values());
          loadedResults.value++;
        });
  }
};

const isInitialLoad = ref(true);

onMounted(async () => {
  await getDeviceTestResults();
  isInitialLoad.value = false;
});

watch(filterConditions, () => {
  if (!isInitialLoad.value) {
    getDeviceTestResults();
  }
}, {deep: true});

/**
 * Refresh the table by fetching the latest emergency light results from the server.
 */
function refreshTable() {
  getDeviceTestResults();
}

/**
 * download the CSV report, fetched from the server
 */
async function downloadCSV() {
  const csvHeaders = headers.map(h => h.title).join(',');
  const getValue = (item, key) => key.split('.').reduce((o, k) => (o ? o[k] : ''), item);

  const csvRows = testResults.value.map(item =>
      headers.map(h => {
        let val;
        if (h.key.startsWith('functionTest') && !item.functionTest) {
          val = '';
        } else if (h.key.startsWith('durationTest') && !item.durationTest) {
          val = '';
        } else {
          val = getValue(item, h.key);
        }
        if (h.key.endsWith('result') && val !== undefined && val !== null && val !== '') {
          val = emergencyLightResultToString(val);
        }
        if (h.key.endsWith('endTime') && val) {
          val = timestampToDate(val).toLocaleString();
        }
        return `"${(val ?? '').toString().replace(/"/g, '""')}"`;
      }).join(',')
  );

  const csvContent = [csvHeaders, ...csvRows].join('\r\n');
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.setAttribute('download', 'emergency_lighting.csv');
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/**
 * @return {Promise<void>}
 */
async function durationTest() {
  const lightingTests = selectedLights.value.map((light) => {
    const req = {
      name: light
    };
    return startDurationTest(req);
  });
  await Promise.all(lightingTests).catch((err) => console.error('Error running test: ', err));
  selectedLights.value = [];
}

/**
 * @return {Promise<void>}
 */
async function functionTest() {
  const lightingTests = selectedLights.value.map((light) => {
    const req = {
      name: light
    };
    return startFunctionTest(req);
  });
  await Promise.all(lightingTests).catch((err) => console.error('Error running test: ', err));
  selectedLights.value = [];
}


</script>

<style scoped></style>