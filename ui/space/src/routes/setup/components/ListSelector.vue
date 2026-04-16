<template>
  <div>
    <h1 class="text-h1 py-12 mt-n12">{{ title }}</h1>
    <v-card class="px-4 py-4" color="rgba(255,255,255,0.3)" :loading="zoneMetadataLoading">
      <v-infinite-scroll
          class="overflow-auto"
          :height="352"
          @load="fetch">
        <template v-if="!noZones">
          <template v-for="item in zoneList" :key="item.id">
            <div class="rounded-xl bg-primary px-4 py-2 my-1" @click="submit(item)">{{ item.title }}</div>
          </template>
        </template>
        <template v-else>
          <v-card-title class="justify-center">No zone available</v-card-title>
        </template>
        <template #empty>
          <hr class="w-75">
        </template>
      </v-infinite-scroll>
    </v-card>
    <v-btn v-if="showSkip" block class="mt-4" variant="text" @click="emits('zone-skipped')">
      Skip
    </v-btn>
    <v-btn v-if="showAdminMode" block class="mt-4" variant="text" @click="emits('admin-mode')">
      Admin Mode
    </v-btn>
    <v-btn v-if="!disableAuthentication" block class="mt-12" variant="text" @click="accountStore.logout">
      Logout
    </v-btn>
  </div>
</template>

<script setup>
import useMetadata from '@/composables/useMetadata';
import useZoneCollection from '@/composables/useZoneCollection';
import {useAccountStore} from '@/stores/account';
import {useConfigStore} from '@/stores/config';
import {useUiConfigStore} from '@/stores/ui-config';
import {storeToRefs} from 'pinia';
import {computed, watch} from 'vue';

const emits = defineEmits(['shouldAutoLogout', 'zone-selected', 'zone-skipped', 'admin-mode']);

defineProps({
  title: {
    type: String,
    default: 'Panel Setup'
  },
  showSkip: {
    type: Boolean,
    default: false
  },
  showAdminMode: {
    type: Boolean,
    default: false
  }
});

const {zoneCollection, getNextZones} = useZoneCollection();
const accountStore = useAccountStore();
const {zones, isInitialized} = storeToRefs(accountStore);
const uiConfig = useUiConfigStore();
const disableAuthentication = computed(() => uiConfig.auth.disabled);
const configStore = useConfigStore();

const zoneIds = computed(() => {
  if (!zoneCollection?.response?.servicesList) {
    return [];
  }
  const allIds = [...new Set(
    zoneCollection.response.servicesList
        .filter(s => s.active)
        .map(s => s.id)
  )];
  if (zones.value.length === 0) return allIds;
  // Filter to zones that exactly match or fall under a configured zone path
  return allIds.filter(id =>
    zones.value.some(z => id === z || id.startsWith(z + '/'))
  );
});
const {loading: zoneMetadataLoading, trackers: zoneMetadata} = useMetadata(zoneIds);

// Combine zone ids with metadata to make the list of zones easier to read - i.e. use titles.
const zoneList = computed(() => {
  return zoneIds.value.map(id => ({
    id,
    metadata: zoneMetadata[id].response,
    title: zoneMetadata[id]?.response?.appearance?.title ?? id
  }));
});

const noZones = computed(() => {
  return zoneList.value.length === 0;
});


const fetch = async ({done}) => {
  if (!isInitialized.value) return; // don't do anything until we know about account zones

  const prev = zoneList.value.length;

  await getNextZones(10);

  if (prev !== 0 && zoneList.value.length === prev) {
    done('empty'); // disable further fetch calls
    return;
  }
  done('ok'); // keep fetching on scroll to bottom of infinite scroll component
};

/**
 *
 * @param {{id: string, metadata: Metadata.AsObject}} zone
 */
function submit(zone) {
  emits('shouldAutoLogout', false);
  emits('zone-selected', zone);
}

// Watch for changes to the zoneList, zoneName, and zoneId
// If zoneName or zoneId is not set, emit an event to auto logout the user
// If a matching zone exists, emit an event to auto logout the user
// If no matching zone exists, do not emit an event to auto logout the user
watch(
    [() => zoneList.value, () => configStore.zoneName, () => configStore.zoneId],
    ([zones, zoneName, zoneId]) => {
      // Check if zoneName or zoneId is not set
      if (!zoneName || !zoneId) {
        emits('shouldAutoLogout', true);
        return;
      }

      // Check if a matching zone exists
      const match = zones.some(zone => zone.text === zoneId);

      emits('shouldAutoLogout', match);
    },
    {immediate: true}
);
</script>