<template>
  <v-autocomplete
      v-model="scopes"
      :items="items"
      v-model:search="searchModel"
      no-filter
      multiple chips closable-chips
      return-object
      label="Scope"
      hide-details>
    <template #chip="{props: chipProps, item}">
      <scope-chip :title="item.raw.title" :value="item.raw.value" :type="item.raw.type" v-bind="chipProps"/>
    </template>
    <template #item="{props: itemProps, item, index}">
      <v-list-subheader v-if="item.raw.header" :title="item.title" :class="{'mt-4': index !== 0}"/>
      <v-list-item v-else-if="item.raw.onClick"
                   v-bind="omit(itemProps, 'onClick')"
                   @click="item.raw.onClick"
                   base-color="info"/>
      <v-list-item v-else-if="item.raw?.type === RoleAssignment.ResourceType.NAMED_RESOURCE" v-bind="itemProps">
        <template #append>
          <v-list-item-action>
            <v-btn icon="mdi-file-tree" variant="plain" v-tooltip:bottom="'Also allow descendants'"
                   @click="toggleNamedDescendants($event, item)"/>
          </v-list-item-action>
        </template>
      </v-list-item>
      <v-list-item v-else v-bind="itemProps" density="compact"/>
    </template>
    <template #append-item v-if="$slots.appendSticky">
      <v-card-actions class="sticky-actions">
        <slot name="appendSticky"/>
      </v-card-actions>
    </template>
  </v-autocomplete>
</template>

<script setup>
import {useDevicesCollection, usePullDevicesMetadata} from '@/composables/devices.js';
import ScopeChip from '@/routes/auth/accounts/ScopeChip.vue';
import {useCohortStore} from '@/stores/cohort.js';
import {RoleAssignment} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {omit} from 'lodash';
import {computed, ref, watch} from 'vue';

const scopes = defineModel({
  type: Array, // of string
  default: () => [],
});
const searchModel = ref(null);
const hasSearchText = computed(() => !!searchModel.value?.trim());

const toggleNamedDescendants = (event, item) => {
  event.stopPropagation();
  const idx = scopes.value.findIndex((i) => i.value === item.value);
  if (idx === -1) {
    scopes.value.push({...item.raw, type: RoleAssignment.ResourceType.NAMED_RESOURCE_PATH_PREFIX});
  } else {
    scopes.value.splice(idx, 1);
  }
}

// for selecting name and name-prefix items
const listDevicesWantCount = ref(10);
const onListDevicesShowMoreClick = () => {
  listDevicesWantCount.value += 10;
}
watch(searchModel, () => listDevicesWantCount.value = 10);
const listDevicesRequest = computed(() => {
  if (!hasSearchText.value) return null; // don't search without text

  const conditions = [];
  const searchStr = searchModel.value;
  const words = searchStr.split(/\s+/);
  for (const word of words) {
    conditions.push({stringContainsFold: word});
  }
  return {query: {conditionsList: conditions}};
});
const {
  items: searchDevices,
  loading: searchDevicesLoading,
  hasMorePages: searchDevicesHasMorePages
} = useDevicesCollection(listDevicesRequest, () => ({wantCount: listDevicesWantCount.value}));

// for selecting zone and subsystem items
const {
  value: devicesMetadata,
  loading: devicesMetadataLoading
} = usePullDevicesMetadata(['metadata.location.zone', 'metadata.membership.subsystem']);

// for selecting node items
const cohortStore = useCohortStore();

const items = computed(() => {
  const items = [];

  items.push({title: 'Global', props: {subtitle: 'All devices and resources', lines: '2'}})

  items.push({title: 'Device', header: true, loading: searchDevicesLoading.value});
  if (!hasSearchText.value) {
    items.push({title: 'Search to find specific devices', props: {disabled: true}});
  } else {
    const deviceToItem = (device) => {
      const title = (() => {
        return device.metadata?.appearance?.title || device.name;
      })()
      const subtitle = (() => {
        if (title === device.name) return null;
        return device.name;
      })()
      const props = {};
      if (subtitle) {
        props.subtitle = subtitle;
        props.lines = '2';
      }
      return {title, type: RoleAssignment.ResourceType.NAMED_RESOURCE, value: device.name, props};
    }
    for (const device of searchDevices.value) {
      items.push(deviceToItem(device));
    }
    if (!searchDevicesLoading.value && searchDevicesHasMorePages.value) {
      items.push({
        title: 'Load more...',
        onClick: onListDevicesShowMoreClick,
        props: {subtitle: 'Or refine your search'}
      });
    }
  }

  const addMdItems = (title, field, type) => {
    items.push({title, header: true, loading: devicesMetadataLoading.value});
    const counts = devicesMetadata.value?.fieldCountsList.find((i) => i.field === field);
    if (!counts) return;
    for (const name of counts.countsMap.map(([k]) => k).sort((a, b) => a.localeCompare(b))) {
      items.push({title: name, type, value: name});
    }
  }
  addMdItems('Zone', 'metadata.location.zone', RoleAssignment.ResourceType.ZONE);
  addMdItems('Subsystem', 'metadata.membership.subsystem', RoleAssignment.ResourceType.SUBSYSTEM);

  items.push({title: 'On Node', header: true, loading: cohortStore.loading.value});
  for (const node of cohortStore.cohortNodes) {
    items.push({title: node.name, type: RoleAssignment.ResourceType.NODE, value: node.name});
  }

  // filter based on search terms
  const matcher = ((str) => {
    if (!str) return () => true;
    const words = str.split(/\s+/).map(word => word.toLowerCase());
    return (item) => {
      if (!item.value) return true;
      if (item.type === RoleAssignment.ResourceType.NAMED_RESOURCE || item.type === RoleAssignment.ResourceType.NAMED_RESOURCE_PATH_PREFIX) {
        return true;
      }
      return words.every((word) => item.title.toLowerCase().includes(word));
    };
  })(searchModel.value);
  return items.filter(matcher);
});
</script>

<style scoped>
:deep(.v-menu .v-list-item + .v-list-subheader) {
  margin-top: 16px;
}

.sticky-actions {
  position: sticky;
  bottom: -8px;
  background-color: rgb(var(--v-theme-surface));
  z-index: 1;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  margin-top: 8px;
}
</style>