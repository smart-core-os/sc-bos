<template>
  <v-toolbar v-if="!props.overviewPage && !props.hideHeader" color="transparent" class="mb-6">
    <v-toolbar-title class="text-h3 ml-0">{{ props.title }}</v-toolbar-title>
    <v-btn color="neutral" @click="doDownloadCSV()" variant="flat">
      Export CSV...
    </v-btn>
  </v-toolbar>

  <component :is="props.overviewPage ? 'div' : VCard" v-bind="tableWrapperProps">
    <v-data-table-server
        :headers="headers"
        :hide-default-header="props.hideTableHeader"
        v-bind="tableAttrs"
        disable-sort
        :items-length="queryTotalCount"
        :row-props="rowProps"
        :class="{ 'hide-pagination': modifyFooter }"
        :hide-default-footer="props.hidePaging"
        @click:row="showNotification">
      <template #top>
        <v-toolbar
            v-if="!props.hideHeader"
            color="transparent"
            extension-height="auto">
          <v-toolbar-title v-if="props.overviewPage" class="text-h4">
            {{ props.title }}
          </v-toolbar-title>
          <v-spacer v-else/>
          <template v-if="!props.hideHeaderActions">
            <filter-choice-chips
                :ctx="filterCtx"
                class="ml-2"/>
            <filter-btn
                :ctx="filterCtx"
                v-bind="btnStyles"/>
            <v-tooltip location="top">
              <template #activator="{ props: _props }">
                <v-btn
                    v-if="props.overviewPage"
                    v-bind="{..._props, ...btnStyles}"
                    @click="doDownloadCSV()">
                  <v-icon size="24">mdi-file-download</v-icon>
                </v-btn>
              </template>
              Export CSV...
            </v-tooltip>
            <v-tooltip location="top">
              <template #activator="{ props: _props }">
                <v-btn
                    v-if="!props.overviewPage"
                    v-bind="{..._props, ...btnStyles}"
                    @click="toggleManualEntry">
                  <v-icon size="30">mdi-plus</v-icon>
                </v-btn>
              </template>
              <span>Add New Entry</span>
            </v-tooltip>
          </template>
          <template #extension v-if="manualEntryPanel !== null">
            <v-expansion-panels v-if="!props.overviewPage" class="mb-3" flat v-model="manualEntryPanel">
              <v-expansion-panel>
                <v-expansion-panel-text>
                  <div class="text-subtitle-2 pl-0 text-body-1 mb-4">Manual Notification Entry Form</div>
                  <v-row class="align-center">
                    <v-col cols="self-align">
                      <v-text-field
                          v-model="manualEntryForm.description"
                          label="Description"
                          density="compact"
                          variant="outlined"
                          hide-details
                          maxlength="160">
                        <template #append-inner>
                          <span class="character-counter">
                            {{ manualEntryForm.description ? 160 - manualEntryForm.description.length : 160 }}
                          </span>
                        </template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="2">
                      <v-select
                          v-model="manualEntryForm.severity"
                          density="compact"
                          hide-details
                          :items="[
                            {label: 'INFO', value: 9},
                            {label:'WARN', value: 13},
                            {label:'ALERT', value:17},
                            {label:'DANGER', value: 21}
                          ]"
                          item-title="label"
                          item-value="value"
                          label="Severity"
                          variant="outlined"/>
                    </v-col>
                    <v-col cols="2">
                      <v-select
                          v-model="manualEntryForm.floor"
                          density="compact"
                          hide-details
                          :items="floors"
                          label="Floor"
                          variant="outlined"/>
                    </v-col>
                    <v-col cols="2">
                      <v-select
                          v-model="manualEntryForm.zone"
                          density="compact"
                          hide-details
                          :items="zones"
                          label="Zone"
                          variant="outlined"/>
                    </v-col>
                    <v-col cols="auto">
                      <v-btn
                          @click="addManualEntry"
                          :disabled="!manualEntryForm.description"
                          color="primary"
                          variant="flat">
                        Add
                      </v-btn>
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>
          </template>
        </v-toolbar>
      </template>
      <template #item.createTime="{ item }">
        {{ timestampToDate(item.createTime).toLocaleString() }}
      </template>
      <template #item.subsystem="{ item }">
        <subsystem-icon size="20px" :subsystem="item.subsystem" no-default/>
      </template>
      <template #item.source="{ item }">
        <v-tooltip location="bottom">
          <template #activator="{ props: _props }">
            <span v-bind="_props">{{ formatSource(item.source) }}</span>
          </template>
          <span>{{ item.source }}</span>
        </v-tooltip>
      </template>
      <template #item.severity="{ item }">
        <v-tooltip v-if="item.resolveTime" location="bottom">
          <template #activator="{ props: _props }">
            <span v-bind="_props">RESOLVED</span>
          </template>
          Was:
          <span :class="severityData(item.severity).color">{{ severityData(item.severity).text }}</span>
        </v-tooltip>
        <span v-else :class="severityData(item.severity).color">{{ severityData(item.severity).text }}</span>
      </template>
      <template #item.acknowledged="{ item }">
        <acknowledgement-btn
            :ack="item.acknowledgement"
            @acknowledge="setAcknowledged(true, item, name)"
            @unacknowledge="setAcknowledged(false, item, name)"/>
      </template>
    </v-data-table-server>
  </component>
</template>
<script setup>
import {timestampToDate} from '@/api/convpb';
import {newActionTracker} from '@/api/resource.js';
import {createAlert} from '@/api/ui/alerts.js';
import FilterBtn from '@/components/filter/FilterBtn.vue';
import FilterChoiceChips from '@/components/filter/FilterChoiceChips.vue';
import useFilterCtx from '@/components/filter/filterCtx.js';
import SubsystemIcon from '@/components/SubsystemIcon.vue';
import {useAlertsCollection} from '@/composables/alerts.js';
import {severityData, useAcknowledgement} from '@/composables/notifications.js';
import {useDataTableCollection} from '@/composables/table.js';
import AcknowledgementBtn from '@/routes/ops/notifications/AcknowledgementBtn.vue';
import {useAlertMetadataStore} from '@/routes/ops/notifications/alertMetadata';
import {downloadCSV} from '@/routes/ops/notifications/export.js';
import {useCohortStore} from '@/stores/cohort.js';
import {useSidebarStore} from '@/stores/sidebar';
import {isNullOrUndef} from '@/util/types.js';
import {useLocalProp} from '@/util/vue.js';
import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb';
import {computed, onUnmounted, reactive, ref} from 'vue';
import {VCard} from 'vuetify/components';

const props = defineProps({
  overviewPage: {
    type: Boolean,
    default: false
  },
  forceQuery: {
    type: Object, /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb').Alert.Query.AsObject} */
    default: null
  },
  title: {
    type: String,
    default: 'Notifications'
  },
  itemPerPage: {
    type: Number,
    default: 20
  },
  hideHeader: {
    type: Boolean,
    default: false
  },
  hidePaging: {
    type: Boolean,
    default: false
  },
  hideTableHeader: {
    type: Boolean,
    default: false
  },
  hideHeaderActions: {
    type: Boolean,
    default: false
  },
  columns: {
    type: Array,
    default: () => []
  },
  fixedRowCount: {
    type: Number,
    default: null
  }
});
const btnStyles = ref({
  'icon': true,
  'tile': true,
  'class': 'rounded ml-2',
  'elevation': 0,
  'size': 'small',
  'variant': 'text'
});

const {setAcknowledged} = useAcknowledgement();
const alertMetadata = useAlertMetadataStore();
const sidebar = useSidebarStore();

const manualEntryValue = reactive(newActionTracker());
const manualEntryPanel = ref(null);
const manualEntryForm = ref({
  source: 'manual',
  description: undefined,
  severity: undefined,
  subsystem: undefined,
  floor: undefined,
  zone: undefined
});
const toggleManualEntry = () => {
  manualEntryPanel.value === null ? manualEntryPanel.value = 0 : manualEntryPanel.value = null;
};

const addManualEntry = async () => {
  await createAlert({alert: manualEntryForm.value}, manualEntryValue);
  manualEntryForm.value = {
    source: 'manual',
    description: undefined,
    severity: undefined,
    subsystem: undefined,
    floor: undefined,
    zone: undefined
  };
};

const modifyFooter = computed(() => queryMetadataCount.value === undefined);

const floors = computed(() => Object.keys(alertMetadata.floorCountsMap)
    .sort((a, b) => a.localeCompare(b, undefined, {numeric: true})));
const zones = computed(() => Object.keys(alertMetadata.zoneCountsMap).sort());
const subsystems = computed(() => Object.keys(alertMetadata.subsystemCountsMap).sort());

const filterOpts = computed(() => {
  // we only add filters that can affect the output, i.e. no floor filter if nothing has a floor.
  const filters = [];
  const defaults = [];
  const forceQuery = props.forceQuery ?? {};
  // Acknowledged
  if (!Object.hasOwn(forceQuery, 'acknowledged')) {
    filters.push({
      key: 'acknowledged',
      icon: 'mdi-checkbox-marked-circle-outline', title: 'Acknowledgement', type: 'boolean',
      valueToString(value) {
        switch (value) {
          case true:
            return 'Acknowledged';
          case false:
            return 'Unacknowledged';
          default:
            return 'All';
        }
      }
    });
  }
  // Floor
  if (!Object.hasOwn(forceQuery, 'floor')) {
    const items = floors.value
        // we can't query for empty strings anyway.
        .filter(s => Boolean(s));
    if (items.length > 0) {
      filters.push({key: 'floor', icon: 'mdi-layers-triple-outline', title: 'Floor', type: 'list', items});
    }
  }
  // Severity
  if (!Object.hasOwn(forceQuery, 'severityNotAbove') &&
      !Object.hasOwn(forceQuery, 'severityNotBelow')) {
    filters.push({
      key: 'severity', // maps to severityNotBelow and severityNotAbove
      icon: 'mdi-alert-box-outline', title: 'Severity', type: 'range',
      items: Object.entries(Alert.Severity)
          .map(([, v]) => ({value: v, title: severityData(v).text}))
          .filter((item) => item.value !== 0) // skip UNSPECIFIED
    });
  }
  // Subsystem
  if (!Object.hasOwn(forceQuery, 'subsystem')) {
    const items = subsystems.value
        // we can't query for empty strings anyway.
        .filter(s => Boolean(s));
    if (items.length > 0) {
      filters.push({key: 'subsystem', icon: 'mdi-file-tree', title: 'Subsystem', type: 'list', items});
    }
  }
  // Zone
  if (!Object.hasOwn(forceQuery, 'zone')) {
    const items = zones.value
        // we can't query for empty strings anyway.
        .filter(s => Boolean(s));
    if (items.length > 0) {
      filters.push({key: 'zone', icon: 'mdi-select-all', title: 'Zone', type: 'list', items});
    }
  }

  if (!Object.hasOwn(forceQuery, 'resolved')) {
    filters.push({
      key: 'resolved',
      icon: 'mdi-checkbox-marked-circle-outline', title: 'Resolution', type: 'boolean',
      valueToString(value) {
        switch (value) {
          case true:
            return 'Resolved';
          case false:
            return 'Unresolved';
          default:
            return 'All';
        }
      }
    });
    defaults.push({filter: 'resolved', value: false});
  }
  return {filters, defaults};
});
const filterCtx = useFilterCtx(filterOpts);

const nonFilterableQueryFields = computed(() => {
  return /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb').Alert.Query.AsObject} */ props.forceQuery ?? {};
});
const queryFields = computed(() => {
  const res = /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb').Alert.Query.AsObject} */ {};
  for (const choice of filterCtx.sortedChoices.value) {
    if (isNullOrUndef(choice?.value)) continue;
    switch (choice.filter) {
      case 'acknowledged':
        res.acknowledged = choice.value;
        break;
      case 'floor':
        res.floor = choice.value?.value ?? choice.value;
        break;
      case 'severity': {
        const {from, to} = choice.value;
        if (from) {
          res.severityNotBelow = from.value;
        }
        if (to) {
          res.severityNotAbove = to.value;
        }
        break;
      }
      case 'subsystem':
        res.subsystem = choice.value?.value ?? choice.value;
        break;
      case 'zone':
        res.zone = choice.value?.value ?? choice.value;
        break;
      case 'resolved':
        res.resolved = choice.value;
        break;
    }
  }
  return res;
});
const query = computed(() => {
  return {...nonFilterableQueryFields.value, ...queryFields.value};
});

const cohort = useCohortStore();
const name = computed(() => cohort.hubNode?.name ?? '');

const alertsRequest = computed(() => ({
  name: name.value,
  query: query.value
}));
const wantCount = useLocalProp(computed(() => props.fixedRowCount || 20));
const alertsOptions = computed(() => ({
  wantCount: wantCount.value
}));
const alertsCollection = useAlertsCollection(alertsRequest, alertsOptions);
const tableOptions = computed(() => {
  return {
    itemsPerPage: props.fixedRowCount || undefined,
  }
});
const tableAttrs = useDataTableCollection(wantCount, alertsCollection, tableOptions);
const tableWrapperProps = computed(() => {
  if (!props.overviewPage) {
    return {
      class: ['px-7', 'py-4']
    }
  } else {
    return {};
  }
});

const queryFieldCount = computed(() => Object.values(query.value).filter((value) => value !== undefined).length);

const doDownloadCSV = () => {
  downloadCSV(alertsRequest.value);
};

/**
 *  Calculate the total number of items in the query
 *
 * @param {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb').Alert.Query.AsObject} query
 * @return {number|undefined}
 */
function calculateQueryMetadataCount(query) {
  const fieldCount = queryFieldCount.value;

  /**
   * Get the total number of alerts for the given severity range
   *
   * @return {number|undefined}
   */
  function getSeverityTotal() {
    let total = 0;
    const severityMax = query.severityNotAbove ?? Infinity;
    const severityMin = query.severityNotBelow ?? -Infinity;
    for (const [level, count] of Object.entries(alertMetadata.severityCountsMap)) {
      if (severityMin <= level && level <= severityMax) total += count;
    }
    return total;
  }

  /**
   * Get the total number of alerts for the given needs attention range
   *
   * @return {number|undefined}
   */
  function getNeedsAttentionTotal() {
    const key = [query.acknowledged ? 'ack' : 'nack', query.resolved ? 'resolved' : 'unresolved'].join('_');
    return alertMetadata.needsAttentionCountsMap[key];
  }

  // Switch on the number of query fields
  switch (fieldCount) {
    case 0: // If there are no query fields, then we can use the total count from the metadata
      return alertMetadata.totalCount;

    case 1: // If there is one query field, then we can use the count from the metadata
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) {
          switch (key) {
            case 'subsystem':
              return alertMetadata.subsystemCountsMap[value];
            case 'floor':
              return alertMetadata.floorCountsMap[value];
            case 'zone':
              return alertMetadata.zoneCountsMap[value];
            case 'acknowledged':
              return alertMetadata.acknowledgedCountMap[value];
            case 'resolved':
              return alertMetadata.resolvedCountMap[value];
            case 'severityNotAbove':
              return getSeverityTotal();
            case 'severityNotBelow':
              return getSeverityTotal();
            default:
              return undefined;
          }
        }
      }
      break;
    case 2: // If there are two or more query fields, then we need to calculate the total ourselves
      if (query.acknowledged !== undefined && query.resolved !== undefined) {
        return getNeedsAttentionTotal();
      }
      if (query.severityNotBelow !== undefined && query.severityNotAbove !== undefined) {
        return getSeverityTotal();
      }
      break;
  }

  return undefined;
}

// Calculate the total number of items in the query
const queryMetadataCount = computed(() => calculateQueryMetadataCount(query.value));
const queryTotalCount = computed(() => {
  const totalCount = queryMetadataCount.value;

  // If the query metadata count is defined, then we can use it
  if (totalCount >= 0) return totalCount;
  // If there is a next page token, then we know there are more pages available.
  else if (alertsCollection.hasMorePages.value) return alertsCollection.items.value.length + 1;
  // If there is no next page token, then we know there are no more pages available.
  else return alertsCollection.items.value.length;
});

const allHeaders = [
  {title: 'Timestamp', value: 'createTime', width: '12em'},
  {value: 'subsystem', width: '20px', class: 'pl-2 pr-0', cellClass: 'pl-2 pr-0'},
  {title: 'Source', value: 'source', width: '15em'},
  {title: 'Floor', value: 'floor', width: '10em'},
  {title: 'Zone', value: 'zone', width: '10em'},
  {title: 'Severity', value: 'severity', width: '8em', align: 'center'},
  {title: 'Description', value: 'description', width: '100%'},
  {title: 'Acknowledged', value: 'acknowledged', align: 'center', width: '12em'}
];
const propHeaders = computed(() => {
  if (props.columns.length === 0) return allHeaders;
  const byValue = allHeaders.reduce((acc, header) => {
    acc[header.value] = header;
    return acc;
  }, {});
  const headers = [];
  for (const column of props.columns) {
    const header = byValue[column];
    if (header) {
      headers.push(header);
    }
  }
  return headers;
});

// We don't include _some_ headers we're filtering out to avoid repetition,
// for example if we're filtering to show Floor1, then all rows would show Floor1 in that column which we don't need to
// see over and over.
const headers = computed(() => {
  const q = query.value;
  return propHeaders.value.filter((header) => {
    if (!['floor', 'zone', 'subsystem', 'source'].includes(header.value)) return true;
    return q[header.value] === undefined;
  });
});

const formatSource = (source) => {
  const parts = source.split('/');
  return parts[parts.length - 1];
};
const rowProps = ({item}) => {
  if (item.resolveTime) return {class: 'resolved'};
  return {};
};

/**
 * Shows the device in the sidebar
 *
 * @param {PointerEvent} e
 * @param {*} item
 */
async function showNotification(e, {item}) {
  sidebar.visible = true;
  sidebar.title = item.source;
  sidebar.data = {metadata: sidebar.data?.metadata, notification: {name, item}};
}

onUnmounted(() => {
  sidebar.closeSidebar();
});
</script>

<style lang="scss" scoped>
:deep(table) {
  table-layout: fixed;
}

:deep(.resolved) {
  color: #fff5 !important;
}

.v-data-table :deep(tr:hover) {
  cursor: pointer;
}

.hide-pagination {
  :deep(.v-data-table-footer__info),
  :deep(.v-pagination__last) {
    display: none;
  }

  :deep(.v-pagination__first) {
    margin-left: 16px;
  }
}

.character-counter {
  font-size: 75%;
  color: rgb(var(--v-theme-primary)); /* Adjust color as needed */
}

.v-data-table {
  :deep(.v-table__wrapper) {
    // Toolbar titles have a leading margin of 20px, table cells have a leading padding of 16px.
    // Correct for this and align the leading edge of text in the first column with the toolbar title.
    padding: 0 4px;
  }
}
</style>
