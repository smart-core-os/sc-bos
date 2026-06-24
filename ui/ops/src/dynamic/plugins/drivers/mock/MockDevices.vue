<template>
  <v-toolbar color="transparent">
    <v-toolbar-title>
      {{ readonly ? 'Viewing' : 'Editing' }} "{{ serviceID }}"
    </v-toolbar-title>
    <v-btn
        v-if="!readonly"
        prepend-icon="mdi-plus"
        text="Add Device"
        variant="text"
        class="mr-2"
        @click="addDevice"/>
    <v-fade-transition>
      <v-btn v-if="refreshNeeded" v-bind="refreshAttrs"
             variant="text" class="mr-4"
             v-tooltip="refreshTooltip">
        Refresh
      </v-btn>
    </v-fade-transition>
    <span v-tooltip="emptyNameCount > 0 ? `${emptyNameCount} device${emptyNameCount === 1 ? '' : 's'} missing a name` : saveTooltip">
      <v-btn v-bind="{...saveAttrs, disabled: saveAttrs.disabled || emptyNameCount > 0, onClick: confirmAndSave}" variant="elevated"/>
    </span>
  </v-toolbar>
  <v-card class="mb-4" max-width="400">
    <v-card-title>Health Check</v-card-title>
    <v-card-text>
      <div class="d-flex align-center gap-4">
        <span class="text-body-2 text-no-wrap">Fault probability</span>
        <v-slider
            v-model="faultProbabilityPct"
            min="0" max="100" step="1"
            :disabled="readonly"
            hide-details/>
        <span class="text-body-2 text-no-wrap" style="min-width: 2.5em">{{ faultProbabilityPct }}%</span>
      </div>
    </v-card-text>
  </v-card>
  <v-card :loading="loading">
    <v-expand-transition>
      <v-alert v-if="alertVisible" v-bind="alertAttrs" tile/>
    </v-expand-transition>
    <v-card-text>
      <v-data-table
          :headers="tableHeaders"
          :items="tableRows"
          :row-props="rowProps"
          :items-per-page="20">
        <template #item.name="{item, value}">
          <v-text-field
              class="text-cell mx-n4"
              :model-value="value" @update:model-value="doUpdateItemName(item, $event)"
              variant="outlined" density="compact"
              hide-details/>
        </template>
        <template #item.traits="{item}">
          <v-autocomplete
              class="text-cell mx-n4"
              :model-value="(item.traits ?? []).map(t => t.name)"
              @update:model-value="doUpdateItemTraits(item, $event)"
              :items="traitItems"
              item-title="title"
              item-value="value"
              multiple
              chips
              closable-chips
              variant="outlined"
              density="compact"
              hide-details/>
        </template>
        <template v-for="slot in textItemSlots" #[slot]="{item, column, value}" :key="slot">
          <v-text-field
              class="text-cell mx-n4"
              :model-value="value" @update:model-value="doUpdateItemText(column.value, item, $event)"
              variant="outlined" density="compact"
              hide-details/>
        </template>
        <template #item._actions="{item}">
          <v-btn
              icon="mdi-tune"
              variant="text"
              density="compact"
              @click="openStateDialog(item)"/>
          <v-btn
              :icon="pendingDeletions.has(item.name) ? 'mdi-delete-restore' : 'mdi-delete'"
              variant="text"
              density="compact"
              :color="pendingDeletions.has(item.name) ? undefined : 'error'"
              :disabled="readonly"
              @click="toggleDelete(item)"/>
        </template>
      </v-data-table>
    </v-card-text>
  </v-card>
  <MockDeviceStateDialog v-model="stateDialogOpen" :device="stateDevice"/>
  <v-dialog v-model="confirmOpen" max-width="400">
    <v-card title="Save changes?">
      <v-card-text>
        <div v-if="pendingSummary.added">{{ pendingSummary.added }} device{{ pendingSummary.added === 1 ? '' : 's' }} added</div>
        <div v-if="pendingSummary.changed">{{ pendingSummary.changed }} device{{ pendingSummary.changed === 1 ? '' : 's' }} changed</div>
        <div v-if="pendingSummary.removed">{{ pendingSummary.removed }} device{{ pendingSummary.removed === 1 ? '' : 's' }} removed</div>
      </v-card-text>
      <v-card-actions>
        <v-spacer/>
        <v-btn text="Cancel" @click="confirmOpen = false"/>
        <v-btn text="Save" color="primary" variant="elevated" @click="doConfirmedSave"/>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import MockDeviceStateDialog from './MockDeviceStateDialog.vue';
import {traitItems} from './traits.js';
import {useServiceConfig} from '@/dynamic/service/service.js';
import {usePopulatedFields} from '@/traits/metadata/metadata.js';
import {get as _get, isEqual, set as _set, unset as _unset} from 'lodash';
import {computed, reactive, ref} from 'vue';

/**
 * @typedef {Metadata.AsObject} Device
 */

const {
  saveAttrs, saveTooltip, doSave,
  refreshAttrs, refreshTooltip, refreshNeeded,
  alertVisible, alertAttrs,
  loading, readonly,
  remoteModel, configModel, serviceID
} = useServiceConfig();
const deviceList = computed(() => /** @type {Device[]} */ configModel.value?.devices ?? []);

const faultProbabilityPct = computed({
  get() {
    return Math.round(((configModel.value?.healthCheck?.faultProbability) ?? 0.15) * 100);
  },
  set(pct) {
    configModel.value = {
      ...configModel.value,
      healthCheck: {...(configModel.value?.healthCheck ?? {}), faultProbability: pct / 100}
    };
  }
});

const populatedMetadataFields = usePopulatedFields(deviceList);

const fieldToHeaderKey = (field) => field.replace(/\./g, '-');
const tableHeaders = computed(() => {
  const dst = /** @type {import('vuetify/lib/components/VDataTable').DataTableHeader[]} */ [
    {title: 'Name', key: 'name', width: '20%'},
    {title: 'Traits', key: 'traits', sortable: false}
  ];

  const partToTitle = (part) => {
    if (part.endsWith('Map')) part = part.slice(0, -3);
    if (part.endsWith('List')) part = part.slice(0, -4);
    return part[0].toUpperCase() + part.slice(1);
  };

  // for nested properties, create nested headers
  const childrenKey = Symbol('header');
  const headersByName = {
    [childrenKey]: dst
  };
  for (const field of populatedMetadataFields.value) {
    const parts = field.split('.');
    let parent = headersByName;
    for (let i = 0; i < parts.length - 1; i++) { // all but the last part
      const part = parts[i];
      if (part.endsWith('Map')) continue; // inline maps
      if (!parent[part]) {
        const children = parent[childrenKey];
        const header = {title: partToTitle(part), children: []};
        children.push(header);
        parent[part] = {[childrenKey]: header.children};
      }
      parent = parent[part];
    }

    // process the last part
    const lastPart = parts[parts.length - 1];
    if (!parent[lastPart]) {
      parent[lastPart] = true;
      parent[childrenKey].push({title: partToTitle(lastPart), key: fieldToHeaderKey(field), value: field});
    }
  }

  dst.push({title: '', key: '_actions', sortable: false, align: 'end'});

  return dst;
});
const tableRows = computed(() => {
  return deviceList.value;
});

const textItemSlots = computed(() => {
  return populatedMetadataFields.value.map(field => 'item.' + fieldToHeaderKey(field));
});

const doUpdateItemText = (key, item, newValue) => {
  const old = _get(item, key);
  if (old === newValue) return;
  if (newValue === '') _unset(item, key);
  else _set(item, key, newValue);
};

const doUpdateItemName = (item, newValue) => {
  if (!newValue) return;
  item.name = newValue;
};

const doUpdateItemTraits = (item, traitNames) => {
  item.traits = traitNames.map(n => ({name: n}));
};

const addDevice = () => {
  const devices = configModel.value?.devices ?? [];
  configModel.value = {...configModel.value, devices: [...devices, {name: '', traits: []}]};
};

const pendingDeletions = reactive(new Set()); // Set<string> of device names

const rowProps = ({item}) => pendingDeletions.has(item.name) ? {class: 'pending-delete'} : {};

const toggleDelete = (item) => {
  if (pendingDeletions.has(item.name)) pendingDeletions.delete(item.name);
  else pendingDeletions.add(item.name);
};

const stateDialogOpen = ref(false);
const stateDevice = ref(null);

const openStateDialog = (item) => {
  stateDevice.value = item;
  stateDialogOpen.value = true;
};

const confirmOpen = ref(false);

const pendingSummary = computed(() => {
  const remoteDevices = remoteModel.value?.devices ?? [];
  const localDevices = (configModel.value?.devices ?? []).filter(d => !pendingDeletions.has(d.name));
  const remoteByName = new Map(remoteDevices.map(d => [d.name, d]));
  const localByName = new Map(localDevices.map(d => [d.name, d]));
  let added = 0, changed = 0, removed = 0;
  for (const [name, local] of localByName) {
    const remote = remoteByName.get(name);
    if (!remote) added++;
    else if (!isEqual(local, remote)) changed++;
  }
  for (const name of remoteByName.keys()) {
    if (!localByName.has(name)) removed++;
  }
  return {added, changed, removed};
});

const emptyNameCount = computed(() =>
  (configModel.value?.devices ?? []).filter(d => !pendingDeletions.has(d.name) && !d.name).length
);

const confirmAndSave = () => {
  if (emptyNameCount.value > 0) return;
  confirmOpen.value = true;
};

const doConfirmedSave = () => {
  const devices = (configModel.value?.devices ?? []).filter(d => !pendingDeletions.has(d.name));
  configModel.value = {...configModel.value, devices};
  pendingDeletions.clear();
  confirmOpen.value = false;
  doSave();
};
</script>

<style scoped>
.text-cell:not(:hover) :deep(.v-field:not(.v-field--focused) .v-field__outline) {
  opacity: 0;
}

:deep(.v-input), :deep(.v-field), :deep(.v-field__field) {
  font-size: inherit;
}

:deep(tr.pending-delete) {
  opacity: 0.5;
  text-decoration: line-through;
}
</style>
