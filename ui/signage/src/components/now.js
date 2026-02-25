import {onBeforeUnmount, onMounted, ref, toValue} from 'vue';

export const MILLISECOND = 1;
export const SECOND = 1000 * MILLISECOND;
export const MINUTE = 60 * SECOND;
export const HOUR = 60 * MINUTE;
export const DAY = 24 * HOUR;

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

  onMounted(() => {
    updateNowWhenNeeded(now.value);
  });
  onBeforeUnmount(() => {
    clearTimeout(handle);
  });

  return {now};
}
