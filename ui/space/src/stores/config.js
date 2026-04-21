import {getMetadata} from '@/api/sc/traits/metadata';
import {useAccountStore} from '@/stores/account.js';
import {useUiConfigStore} from '@/stores/ui-config.js';
import {defineStore} from 'pinia';
import {computed, ref} from 'vue';
import {useRouter} from 'vue-router';

export const useConfigStore = defineStore('config', () => {
  // Single zone — the panel's own room. Persisted as 'zoneId' for backward compat.
  const zoneId = ref('');
  const zoneMeta = ref({});

  // Joined zone — the combined space when the rooms are merged (optional).
  const joinedZoneId = ref('');
  const joinedZoneMeta = ref({});

  // Current display mode: 'single' (own room) or 'joined' (combined room).
  const mode = ref('single');

  // The zone currently active for the home page widgets.
  const activeZoneId = computed(() =>
    mode.value === 'joined' && joinedZoneId.value ? joinedZoneId.value : zoneId.value);
  const activeZoneMeta = computed(() =>
    mode.value === 'joined' && joinedZoneId.value ? joinedZoneMeta.value : zoneMeta.value);

  const hasJoinedZone = computed(() => Boolean(joinedZoneId.value));

  const zoneName = computed(() => activeZoneMeta.value?.appearance?.title ?? activeZoneId.value ?? '');

  const isAdminMode = ref(false);

  const isReconfiguring = ref(false);
  const isConfigured = computed(() => {
    return Boolean(zoneId.value) || isAdminMode.value;
  });

  /**
   * @param {string} zone
   * @param {Metadata.AsObject} [meta]
   */
  async function setZone(zone, meta = null) {
    if (zone) {
      isReconfiguring.value = false;
      isAdminMode.value = false;
      zoneId.value = zone;
      if (meta) {
        zoneMeta.value = meta;
      } else {
        zoneMeta.value = await getMetadata({name: zone});
      }
    }
  }

  /**
   * @param {string} zone - empty string to clear the joined zone
   * @param {Metadata.AsObject} [meta]
   */
  async function setJoinedZone(zone, meta = null) {
    joinedZoneId.value = zone ?? '';
    if (zone) {
      joinedZoneMeta.value = meta ? meta : await getMetadata({name: zone});
    } else {
      joinedZoneMeta.value = {};
    }
  }

  /**
   * Toggle between 'single' and 'joined' mode. No-op if no joined zone is configured.
   */
  function toggleMode() {
    if (!hasJoinedZone.value) return;
    mode.value = mode.value === 'single' ? 'joined' : 'single';
  }

  /**
   *
   */
  function setAdminMode() {
    isAdminMode.value = true;
    isReconfiguring.value = false;
  }

  /**
   *
   */
  function reset() {
    zoneId.value = '';
    zoneMeta.value = {};
    joinedZoneId.value = '';
    joinedZoneMeta.value = {};
    mode.value = 'single';
    isAdminMode.value = false;
  }

  const uiConfig = useUiConfigStore();
  const router = useRouter();
  const accountStore = useAccountStore();

  /**
   * Causes the panel to enter the set-up flow, even when it's already set up.
   */
  function reconfigure() {
    // if (isReconfiguring.value) return;
    isReconfiguring.value = true;

    if (uiConfig.auth.disabled) {
      router.push({name: 'setup'}).catch(() => {});
    } else {
      accountStore.forceLogIn = true; // cleared on page reload
      router.push({name: 'login'}).catch(() => {});
    }
  }

  /**
   * Called when a reconfigure should be aborted, for example when clicking "back to home"
   */
  function abortReconfigure() {
    isReconfiguring.value = false;
    accountStore.forceLogIn = false;
    router.push('/').catch(() => {});
  }

  return {
    zoneId,
    zoneMeta,
    joinedZoneId,
    joinedZoneMeta,
    activeZoneId,
    activeZoneMeta,
    hasJoinedZone,
    mode,
    zoneName,
    isAdminMode,
    isConfigured,
    isReconfiguring,
    setZone,
    setJoinedZone,
    setAdminMode,
    toggleMode,
    reset,
    reconfigure,
    abortReconfigure,
  };
}, {persist: true});
