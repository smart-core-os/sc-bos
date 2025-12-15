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
      <v-btn
          color="primary"
          class="ml-2"
          :disabled="loadedResults < totalDevices"
          @click="downloadCSV">
        Download CSV
        <v-icon end>mdi-download</v-icon>
      </v-btn>
    </v-card-title>
    <v-data-table
        :headers="headers"
        :items="testResults"
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
  emergencyLightResultToString,
  getTestResultSet,
  startDurationTest,
  startFunctionTest
} from '@/api/sc/traits/emergency-light.js';
import {listDevices} from '@/api/ui/devices.js';
import ContentCard from '@/components/ContentCard.vue';
import {batchLargeArray, mapLargeArray} from '@/util/array.js';
import {computed, onMounted, ref} from 'vue';

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

const findEmLightsQuery = computed(() => {
  const q = {conditionsList: []};
  q.conditionsList.push({field: 'metadata.traits.name', stringEqual: 'smartcore.bos.EmergencyLight'});
  return q;
});

const getDeviceTestResults = async () => {
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

  totalDevices.value = allDevices.length;

  for (const item of allDevices) {
    getTestResultSet({name: item.name, queryDevice: true})
        .then(testResult => {
          testResults.value.push({
            name: item.name,
            functionTest: testResult.functionTest,
            durationTest: testResult.durationTest
          });
          loadedResults.value++;
        })
        .catch(err => {
          console.error('Error fetching test results for device: ', item.name, err);
          testResults.value.push({
            name: item.name,
            functionTest: {
              testResult: -1,
            },
            durationTest: {
              testResult: -1,
            }
          });
          loadedResults.value++;
        });
  }
};

onMounted(async () => {
  await getDeviceTestResults();
});

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

  const csvRows = mapLargeArray(batchLargeArray(testResults), item =>
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
      }).join(','), true);

  const csvContent = [csvHeaders, ...csvRows].join('\r\n');
  const blob = new Blob([csvContent], {type: 'text/csv;charset=utf-8;'});
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