import {getMetadata} from '@/api/sc/traits/metadata';
import {useAccountStore} from '@/stores/account.js';
import {useUiConfigStore} from '@/stores/ui-config.js';
import {defineStore} from 'pinia';
import {computed, ref} from 'vue';
import {useRouter} from 'vue-router';

export const useConfigStore = defineStore('config', () => {
  const zoneId = ref('');
  const zoneMeta = ref({});

  const zoneName = computed(() => zoneMeta.value?.appearance?.title ?? zoneId.value ?? '');

  const isReconfiguring = ref(false);
  const isConfigured = computed(() => {
    return Boolean(zoneId.value);
  });

  /**
   * @param {string} zone
   * @param {Metadata.AsObject} [meta]
   */
  async function setZone(zone, meta = null) {
    if (zone) {
      isReconfiguring.value = false;
      zoneId.value = zone;
      if (meta) {
        zoneMeta.value = meta;
      } else {
        zoneMeta.value = await getMetadata({name: zone});
      }
    }
  }

  /**
   *
   */
  function reset() {
    zoneId.value = '';
    zoneMeta.value = {};
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
    zoneName,
    isConfigured,
    isReconfiguring,
    setZone,
    reset,
    reconfigure,
    abortReconfigure,
  };
}, {persist: true});
