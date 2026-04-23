import {MINUTE} from '@/util/date.js';
import {onScopeDispose, ref, toValue} from 'vue';

/**
 *
 * @param {MaybeRefOrGetter<number>} resolution
 * @return {{now: import('vue').Ref<Date>}}
 */
export function useNow(resolution = MINUTE) {
  const now = ref(new Date());

  /**
   *
   * @param {Date} t
   * @return {number}
   */
  function nextDelay(t) {
    const ms = t.getTime();
    const res = toValue(resolution);
    // note: if ms is exactly on a resolution boundary then instead of returning 0 we should wait a full hop
    return (ms % res) || res;
  }

  let handle = 0;

  /**
   *
   * @param {Date} t
   */
  function updateNowWhenNeeded(t) {
    const delay = nextDelay(t);
    clearTimeout(handle);
    handle = setTimeout(() => {
      now.value = new Date();
      updateNowWhenNeeded(now.value);
    }, delay);
  }

  updateNowWhenNeeded(now.value);
  onScopeDispose(() => {
    clearTimeout(handle);
  });

  return {now};
}
