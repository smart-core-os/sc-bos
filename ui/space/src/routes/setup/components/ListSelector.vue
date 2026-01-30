<template>
  <div>
    <h1 class="text-h1 py-12 mt-n12">Panel Setup</h1>
    <v-card class="px-4 py-4" color="rgba(255,255,255,0.3)" :loading="zoneMetadataLoading">
      <template v-if="!noZones">
        <v-infinite-scroll
            class="overflow-auto"
            :height="352"
            mode="manual"
            @load="fetch">
          <template v-for="item in zoneList" :key="item.id">
            <div class="rounded-xl bg-primary px-4 py-2 my-1" @click="submit(item)">{{ item.title }}</div>
          </template>
          <template #empty>
            <div class="text-center py-2 text-caption">No more zones</div>
          </template>
        </v-infinite-scroll>
      </template>
      <template v-else>
        <v-card-title class="justify-center">No zone available</v-card-title>
      </template>
    </v-card>
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
import {computed, ref, watch} from 'vue';
import {useRouter} from 'vue-router';

const emits = defineEmits(['shouldAutoLogout']);

const router = useRouter();
const {zoneCollection, getNextZones, hubNodesValue, enrollmentValue} = useZoneCollection();
const accountStore = useAccountStore();
const {zones, isInitialized} = storeToRefs(accountStore);
const uiConfig = useUiConfigStore();
const disableAuthentication = computed(() => uiConfig.auth.disabled);
const configStore = useConfigStore();

const zoneIds = computed(() => {
  // prefer using the account zones over fetching from the server
  if (zones.value.length > 0) return zones.value;
  if (!zoneCollection?.response?.servicesList) {
    return [];
  }
  const zoneIds = zoneCollection.response.servicesList
      .filter(s => s.active)
      .map(s => s.id);
  return [...new Set(zoneIds)];
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

const noZones = ref(false);


const fetch = async ({done}) => {
  if (!isInitialized.value) {
    // don't do anything until we know about account zones
    done('ok');
    return;
  }
  // keep fetching until hub nodes and enrollment are loaded
  if (hubNodesValue.loading || enrollmentValue.loading) {
    done('ok');
    return;
  }
  if (zones.value.length > 0) {
    // don't load zones from server if we have account zones to use
    done('empty');
    return;
  }

  const fetched = await getNextZones(10);

  if (fetched === null) {
    noZones.value = zoneCollection.response.serviceList.length === 0;
    done('empty'); // disable further fetch calls
    return;
  }
  done('ok'); // keep fetching on scroll to bottom of infinite scroll component
};

/**
 *
 * @param {{id: string, metadata: Metadata.AsObject}} zone
 */
async function submit(zone) {
  await configStore.setZone(zone.id, zone.metadata);
  emits('shouldAutoLogout', false);
  await router.push({name: 'home'});
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