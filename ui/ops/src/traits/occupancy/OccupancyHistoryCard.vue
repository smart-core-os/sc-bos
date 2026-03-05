<template>
  <v-menu :close-on-content-click="false">
    <template #activator="{ props }">
      <v-btn :icon="true" elevation="0" size="x-small" @click="resetMenu" v-bind="props" class="mt-n1 mr-n1">
        <v-icon size="20">mdi-dots-vertical</v-icon>
      </v-btn>
    </template>
    <v-card class="history-card" min-width="360">
      <v-card-text>
        <div class="d-flex align-start">
          <v-date-input
              v-model="dateRange" multiple="range" :readonly="fetchingHistory"
              label="Download History" placeholder="from - to" persistent-placeholder prepend-icon=""
              hint="Select a date range to download historical data." persistent-hint
              :error-messages="downloadError"/>
          <div v-tooltip="downloadBtnDisabled || 'Download CSV...'">
            <v-btn
                @click="onDownloadClick()"
                icon="mdi-file-download" elevation="0" class="ml-2 mr-n2 mt-1"
                :loading="fetchingHistory" :disabled="!!downloadBtnDisabled"/>
          </div>
        </div>
        <div class="mt-2">
          <div class="text-caption text-medium-emphasis mb-1">Columns</div>
          <v-checkbox-btn
              v-for="field in optionalFields" :key="field.name"
              v-model="selectedFields" :value="field.name" :label="field.title"
              density="compact" hide-details/>
        </div>
      </v-card-text>
    </v-card>
  </v-menu>
</template>

<script setup>
import {triggerDownload} from '@/components/download/download.js';
import {addDays, startOfDay} from 'date-fns';
import {computed, ref} from 'vue';
import {VDateInput} from 'vuetify/labs/components';

const p = defineProps({
  name: {
    type: String,
    required: true
  }
});

const optionalFields = [
  {name: 'occupancy.state', title: 'State'},
  {name: 'occupancy.peoplecount', title: 'People Count'},
];

const dateRange = ref([]);
const startDate = computed(() => dateRange.value[0]);
const endDate = computed(() => dateRange.value[dateRange.value.length - 1]);
const selectedFields = ref(optionalFields.map(f => f.name));
const downloadBtnDisabled = computed(() => {
  if (dateRange.value.length === 0) {
    return 'No date range selected';
  }
  if (selectedFields.value.length === 0) {
    return 'No columns selected';
  }
  return '';
});
const fetchingHistory = ref(false);
const downloadError = ref('');

/**
 * Resets the menu to its initial state.
 */
function resetMenu() {
  dateRange.value = [];
  downloadError.value = '';
  selectedFields.value = optionalFields.map(f => f.name);
}

const onDownloadClick = async () => {
  if (!p.name || p.name === '') {
    downloadError.value = 'No device name provided';
    return;
  }
  try {
    fetchingHistory.value = true;
    const names = [p.name];
    const mandatory = [
      {name: 'timestamp', title: 'Reading Time'},
      {name: 'md.name', title: 'Device Name'},
    ];
    const selected = optionalFields.filter(f => selectedFields.value.includes(f.name));
    await triggerDownload(
        'occupancy',
        {conditionsList: [{stringIn: {stringsList: names}}]},
        {startTime: startOfDay(startDate.value), endTime: startOfDay(addDays(endDate.value, 1))},
        {includeColsList: [...mandatory, ...selected]}
    )
  } finally {
    fetchingHistory.value = false;
  }
}
</script>
