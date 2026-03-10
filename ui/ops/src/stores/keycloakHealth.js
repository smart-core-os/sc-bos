import {usePoll} from '@/composables/poll.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {defineStore} from 'pinia';
import {computed, ref, watch} from 'vue';

export const useKeycloakHealthStore = defineStore('keycloakHealth', () => {
  const uiConfig = useUiConfigStore();

  const keycloakConfig = computed(() => uiConfig.auth.keycloak);
  const isConfigured = computed(() => Boolean(keycloakConfig.value));

  const pending = ref(true);
  const error = ref(/** @type {{code: number, message: string}|null} */ null);

  const checkHealth = async () => {
    const cfg = keycloakConfig.value;
    if (!cfg) {
      error.value = null;
      pending.value = false;
      return;
    }

    pending.value = true;
    try {
      const base = cfg.url.endsWith('/') ? cfg.url : cfg.url + '/';
      const url = `${base}realms/${cfg.realm}/.well-known/openid-configuration`;
      const res = await fetch(url);
      if (!res.ok) {
        error.value = {code: res.status, message: `Unable to reach auth provider at ${cfg.url}`};
      } else {
        error.value = null;
      }
    } catch {
      error.value = {code: 0, message: `Unable to reach auth provider at ${cfg.url}`};
    } finally {
      pending.value = false;
    }
  };

  const {lastPoll, nextPoll, pollNow, isPolling} = usePoll(checkHealth);

  // When keycloak config becomes available (async config load), re-trigger the check.
  // Without this, the initial checkHealth() runs with no config, sets pending=false,
  // and the UI shows "reachable" until the next scheduled poll fires.
  watch(keycloakConfig, (cfg, oldCfg) => {
    if (cfg && !oldCfg) {
      pending.value = true;
      pollNow();
    }
  });

  const url = computed(() => keycloakConfig.value?.url ?? '');
  const realm = computed(() => keycloakConfig.value?.realm ?? '');

  const chipColor = computed(() => {
    if (!isConfigured.value) return 'neutral-lighten-1';
    if (error.value) return 'error';
    return 'neutral-lighten-1';
  });

  const tooltipText = computed(() => {
    if (pending.value) return 'Checking auth provider...';
    if (error.value) return error.value.message;
    return `Auth provider reachable at ${url.value}`;
  });

  return {
    isConfigured,
    pending,
    error,
    url,
    realm,
    chipColor,
    tooltipText,
    pollNow,
    lastPoll,
    nextPoll,
    isPolling
  };
});
