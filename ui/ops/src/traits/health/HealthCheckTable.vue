<template>
  <v-data-table-server
      v-bind="tableAttrs"
      v-model:expanded="expanded"
      :headers="headers"
      item-value="name"
      show-expand
      no-data-text="No health checks">
    <template #item.device="{ item }">
      <md-text :value="item.metadata"/>
    </template>
    <template #item.normality="{ item }">
      <normality-last-change-cell :model-value="item"/>
    </template>
    <template #item.reliability="{ item }">
      <reliability-last-change-cell :model-value="item"/>
    </template>
    <template #item.totalCount="{ item }">
      <check-count-cell v-bind="countChecks(item.healthChecksList)"/>
    </template>
    <template #item.data-table-expand="{ internalItem, isExpanded, toggleExpand }">
      <v-btn
          :append-icon="isExpanded(internalItem) ? 'mdi-chevron-up' : 'mdi-chevron-down'"
          :text="isExpanded(internalItem) ? 'Collapse' : 'More info'"
          class="text-none"
          color="medium-emphasis"
          size="small"
          variant="text"
          width="105"
          border
          slim
          @click="toggleExpand(internalItem)"/>
    </template>
    <template #expanded-row="{ item, columns }">
      <tr>
        <td :colspan="columns.length" class="py-2">
          <v-sheet rounded border color="transparent">
            <v-table density="compact" class="bg-transparent checks-table">
              <thead class="bg-surface">
                <tr>
                  <th class="checks-table--check">Check</th>
                  <th class="checks-table--health">Health</th>
                  <th class="checks-table--connection">Connection</th>
                  <th class="checks-table--value">Value</th>
                  <th class="checks-table--impact">Impact</th>
                </tr>
              </thead>
              <tbody>
                <health-check-enriched-rows
                    :device-name="item.name"
                    :health-checks="item.healthChecksList"/>
              </tbody>
            </v-table>
          </v-sheet>
        </td>
      </tr>
    </template>
  </v-data-table-server>
</template>

<script setup>
import MdText from '@/components/MdText.vue';
import {useDevices} from '@/composables/devices.js';
import {useDataTableCollection} from '@/composables/table.js';
import CheckCountCell from '@/traits/health/CheckCountCell.vue';
import {countChecks} from '@/traits/health/health';
import HealthCheckEnrichedRows from '@/traits/health/HealthCheckEnrichedRows.vue';
import NormalityLastChangeCell from '@/traits/health/NormalityLastChangeCell.vue';
import ReliabilityLastChangeCell from '@/traits/health/ReliabilityLastChangeCell.vue';
import {computed, ref} from 'vue';

const props = defineProps({
  conditions: {
    type: Array, // of Device.Query.Condition.AsObject
    default: () => ([
      {'field': 'health_checks.normality', 'stringIn': {'stringsList': ['ABNORMAL', 'HIGH', 'LOW']}}
    ])
  }
});

const wantCount = ref(20);
const _useDevicesOpts = computed(() => {
  return {
    conditions: props.conditions,
    wantCount: wantCount.value
  }
});
const devices = useDevices(_useDevicesOpts);
const tableAttrs = useDataTableCollection(wantCount, devices);
const headers = computed(() => {
  return [
    {title: 'Device', key: 'device'},
    {title: 'Health', key: 'normality'},
    {title: 'Connection', key: 'reliability'},
    {title: 'Issue Count', key: 'totalCount', align: 'end'}
  ]
})

// Only allow one expanded row at a time
const expandedRow = ref(null);
const expanded = computed({
  get() {
    return expandedRow.value ? [expandedRow.value] : [];
  },
  set(value) {
    expandedRow.value = value.length > 0 ? value[value.length - 1] : null;
  }
});
</script>

<style scoped>
.v-data-table {
  background: transparent;
}

.checks-table :deep(table) {
  table-layout: fixed;
}

.checks-table--check {
  width: 40%;
}
</style>