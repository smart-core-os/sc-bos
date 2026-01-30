import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed} from 'vue';

/**
 * Returns a computed ref indicating whether the given experiment is enabled.
 *
 * @param {string} name
 * @return {ComputedRef<boolean>}
 */
export function useExperiment(name) {
  const uiConfigStore = useUiConfigStore();
  return computed(() => Boolean(uiConfigStore.experimentsGetOrDefault(name, false)))
}
