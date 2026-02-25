<template>
  <content-card>
    <v-toolbar color="transparent" class="pl-2 py-2">
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
      <device-download content="icon" :query="devices.query.value" class="ml-2"/>
    </v-toolbar>
    <v-data-table-server
        v-model="selectedDevicesComp"
        v-bind="tableAttrs"
        :headers="headers"
        item-key="name"
        :row-props="rowProps"
        :items-per-page-options="[
          {title: '20', value: 20},
          {title: '50', value: 50},
          {title: '100', value: 100}
        ]"
        :show-select="showSelect"
        item-value="name"
        :class="tableClasses"
        :fixed-header="props.fixedHeader"
        :style="tableStyles"
        @click:row="showDevice">
      <template #item.metadata.membership.subsystem="{ item }">
        <subsystem-icon size="20px" :subsystem="item.metadata?.membership?.subsystem" no-default/>
      </template>
      <template #item.name="{ item }">
        {{ item.metadata.appearance ? item.metadata.appearance.title : item.name }}
      </template>
      <template #item.hotpoint="{item}">
        <hot-point
            v-slot="{live}"
            class="d-flex align-center justify-end"
            :item-key="item.name"
            style="height:100%">
          <device-cell :paused="!live" :item="item"/>
        </hot-point>
      </template>
    </v-data-table-server>
  </content-card>
</template>

<script setup>
import ContentCard from '@/components/ContentCard.vue';
import DeviceDownload from '@/components/download/DeviceDownloadBtn.vue';
import FilterBtn from '@/components/filter/FilterBtn.vue';
import FilterChoiceChips from '@/components/filter/FilterChoiceChips.vue';
import HotPoint from '@/components/HotPoint.vue';
import SubsystemIcon from '@/components/SubsystemIcon.vue';
import {useDeviceFilters, useDevices} from '@/composables/devices';
import {useDataTableCollection} from '@/composables/table.js';
import DeviceSideBar from '@/routes/devices/components/DeviceSideBar.vue';
import {useSidebarStore} from '@/stores/sidebar';
import {computed, ref} from 'vue';
import DeviceCell from './DeviceCell.vue';

const props = defineProps({
  subsystem: {
    type: String,
    default: undefined
  },
  showSelect: {
    type: Boolean,
    default: false
  },
  rowSelect: {
    type: Boolean,
    default: true
  },
  selectedDevices: {
    type: Array,
    default: () => []
  },
  filter: {
    type: Function,
    default: () => true
  },
  forceQuery: {
    type: Object, // keys are condition properties, values are stringEqualFold values
    default: () => ({})
  },
  fixedHeader: {
    type: Boolean,
    default: true
  },
  height: {
    type: [String, Number],
    default: undefined
  }
});
const emit = defineEmits(['update:selectedDevices']);

const sidebar = useSidebarStore();

// searching and filtering
const search = ref('');
const forcedFilters = computed(() => {
  const res = {};
  Object.assign(res, props.forceQuery ?? {}); // default values, explicit props override this
  if (props.subsystem) {
    res['metadata.membership.subsystem'] = props.subsystem === 'all' ? undefined : props.subsystem;
  }
  return res;
});
const {filterCtx, forcedConditions, filterConditions} = useDeviceFilters(forcedFilters);
const hasFilters = computed(() => filterCtx.filters.value.length > 0);

// pagination
const wantCount = ref(20); // same as initial itemsPerPage

const _useDevicesOpts = computed(() => {
  return {
    filter: props.filter,
    search: search.value,
    conditions: [...forcedConditions.value, ...filterConditions.value],
    wantCount: wantCount.value
  };
});
const devices = useDevices(_useDevicesOpts);
const tableAttrs = useDataTableCollection(wantCount, devices);
const {items} = devices;

const headers = ref([
  {key: 'metadata.membership.subsystem', width: '20px', class: 'pl-4 pr-0', cellClass: 'pl-4 pr-0', sortable: false},
  {title: 'Device name', key: 'name'},
  {title: 'Floor', key: 'metadata.location.floor'},
  {title: 'Zone', key: 'metadata.location.zone'},
  {key: 'hotpoint', align: 'end', width: '100', sortable: false}
]);

const tableClasses = computed(() => {
  const c = [];
  if (props.showSelect) c.push('selectable');
  if (props.rowSelect) c.push('rowSelectable');
  return c.join(' ');
});

const tableStyles = computed(() => {
  if (!props.fixedHeader) {
    return {};
  }
  if (props.height) {
    return {height: typeof props.height === 'number' ? `${props.height}px` : props.height};
  }
  return {height: '80vh'};
});

const selectedDevicesComp = computed({
  get() {
    return items.value.filter(device => props.selectedDevices.indexOf(device.name) >= 0);
  },
  set(value) {
    emit('update:selectedDevices', value);
  }
});

/**
 * Shows the device in the sidebar
 *
 * @param {PointerEvent} e
 * @param {*} item
 */
function showDevice(e, {item}) {
  sidebar.visible = true;
  sidebar.title = item.metadata.appearance ? item.metadata.appearance.title : item.name;
  sidebar.data = item;
  sidebar.component = DeviceSideBar; // this line must be after the .data one!
}

/**
 * @param {*} item
 * @return {Record<string, any>}
 */
function rowProps({item}) {
  if (sidebar.visible && sidebar.data?.name === item.name) {
    return {class: 'item-selected'};
  }
  return {};
}
</script>

<style lang="scss" scoped>
@use 'vuetify/settings';

:deep(.v-data-table-header__icon) {
  margin-left: 8px;
}

.v-data-table :deep(.v-data-footer) {
  background: rgb(var(--v-theme-neutral-lighten-1)) !important;
  border-radius: 0 0 settings.$border-radius-root*2 settings.$border-radius-root*2;
  border: none;
  margin: 0 -12px -12px;
}


.v-data-table:not(.selectable) :deep(.v-data-table__selected) {
  background: none;
}

.v-data-table.rowSelectable :deep(.item-selected) {
  background-color: rgb(var(--v-theme-primary-darken-4));
}
</style>
