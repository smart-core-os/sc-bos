import {MINUTE} from '@/components/now.js';
import {computed, onScopeDispose, ref, toValue, watch} from 'vue';

/**
 * Returns an older version of value that is no older than age.
 *
 * @param {import('vue').MaybeRefOrGetter<T>} value
 * @param {number} [age] - maximum age of history, default 5 minutes
 * @param {number} [resolution] - resolution to sample history, default 1 minute. Updates to value more frequent than this will be ignored.
 * @return {import('vue').ComputedRef<T>}
 * @template T
 */
export function useRollingHistory(value, age = 5 * MINUTE, resolution = MINUTE) {
  /**
   * @typedef {Object} Record
   * @property {number} t - timestamp in milliseconds
   * @property {T} v - value
   */
  /** @type {import('vue').Ref<Record[]>} */
  const oldValues = ref([]);
  const lastRecordedTime = ref(0);
  watch(() => toValue(value), (newValue) => {
    const now = Date.now();
    if (lastRecordedTime.value - now < resolution) {
      lastRecordedTime.value = now;
      oldValues.value.push({t: now, v: newValue});
    }
  }, {deep: true});
  let timer = 0;
  onScopeDispose(() => clearTimeout(timer));
  const processOldValues = () => {
    const now = Date.now();
    while (oldValues.value.length > 0 && (oldValues.value[0].t - now) < age) {
      oldValues.value.shift();
    }
  }
  watch(oldValues, (vs) => {
    clearTimeout(timer);
    if (vs.length <= 1) return;
    timer = setTimeout(() => {
      processOldValues();
    }, age - (vs[1].t - Date.now()))
  })

  return {
    oldValue: computed(() => {
      return oldValues.value?.[0]?.v ?? toValue(value);
    }),
    oldValues,
  };
}