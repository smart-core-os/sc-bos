<template>
  <div>
    <v-toolbar color="transparent" class="pl-2 py-2 mb-2">
      <v-text-field
          v-model="search"
          append-inner-icon="mdi-magnify"
          label="Search devices"
          hide-details
          variant="filled"
          max-width="600px"
          class="flex-fill mr-auto"/>
      <template v-if="hasFilters">
        <filter-choice-chips :ctx="filterCtx" class="mx-2"/>
        <filter-btn :ctx="filterCtx"/>
      </template>
    </v-toolbar>
    <v-data-table-server
        v-bind="tableAttrs"
        v-model:expanded="expanded"
        :headers="headers"
        item-value="name"
        show-expand
        no-data-text="No health checks">
      <template #item.device="{ item }">
        <div class="d-flex align-center gap-2">
          <md-text :value="item.metadata"/>
          <v-btn
              icon="mdi-information-outline"
              size="x-small"
              variant="text"
              v-tooltip:bottom="'View UDMI points'"
              @click.stop="openUdmiDialog(item)"/>
        </div>
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
    <v-dialog v-model="udmiDialog.open" max-width="600">
      <udmi-card :name="udmiDialog.device"/>
    </v-dialog>
  </div>
</template>

<script setup>
import FilterBtn from '@/components/filter/FilterBtn.vue';
import FilterChoiceChips from '@/components/filter/FilterChoiceChips.vue';
import MdText from '@/components/MdText.vue';
import {ABNORMAL_NORMALITIES, unhealthyDeviceConditions, useDevices} from '@/composables/devices.js';
import {useDataTableCollection} from '@/composables/table.js';
import CheckCountCell from '@/traits/health/CheckCountCell.vue';
import {countChecks, useHealthCheckFilters} from '@/traits/health/health';
import HealthCheckEnrichedRows from '@/traits/health/HealthCheckEnrichedRows.vue';
import NormalityLastChangeCell from '@/traits/health/NormalityLastChangeCell.vue';
import ReliabilityLastChangeCell from '@/traits/health/ReliabilityLastChangeCell.vue';
import UdmiCard from '@/traits/udmi/UdmiCard.vue';
import {computed, ref} from 'vue';

// The rows shown when neither conditions nor unhealthy is set: checks reporting a fault.
const ABNORMAL_CONDITIONS = [
  {'field': 'health_checks.normality', 'stringIn': {'stringsList': ABNORMAL_NORMALITIES}}
];

const props = defineProps({
  /**
   * Which devices to list, as raw query conditions. Overrides the default selection
   * outright, including the one unhealthy asks for.
   */
  conditions: {
    type: Array, // of Device.Query.Condition.AsObject
    default: null
  },
  /**
   * List devices the way useDeviceHealthCount counts them: any check abnormal *or*
   * unreadable, rather than the abnormal-only default.
   *
   * Worth setting wherever this table sits next to a MeterHealthCard. The two disagree
   * otherwise - a comms failure leaves normality NORMAL, so a meter the card counts as not
   * reporting never appears in the table - and a dashboard showing "654 / 680 reporting"
   * above an empty table reads as a bug in the table.
   */
  unhealthy: {type: Boolean, default: false},
  /**
   * A fully-qualified trait name, or several, limiting the table to devices implementing it
   * (any of them, if given a list).
   *
   * This composes with conditions rather than replacing it, so a config wanting unhealthy
   * meters sets trait alone and leaves the selection to its default. Setting conditions
   * still overrides that default outright.
   */
  trait: {type: [String, Array], default: null}
});

// conditions wins if given, so an explicit [] still clears the filter entirely.
const selection = computed(() => {
  if (props.conditions) return props.conditions;
  return props.unhealthy ? unhealthyDeviceConditions() : ABNORMAL_CONDITIONS;
});

// Filter setup
const search = ref('');
const {filterCtx, filterConditions} = useHealthCheckFilters();
const hasFilters = computed(() => filterCtx.filters.value.length > 0);

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

const udmiDialog = ref({open: false, device: ''});

function openUdmiDialog(item) {
  udmiDialog.value.device = item.name;
  udmiDialog.value.open = true;
}

const wantCount = ref(20);
const _useDevicesOpts = computed(() => {
  return {
    search: search.value,
    conditions: [...selection.value, ...filterConditions.value],
    trait: props.trait,
    wantCount: wantCount.value,
    paused: expandedRow.value !== null,
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
