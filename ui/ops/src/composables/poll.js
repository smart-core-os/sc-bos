import {useNow} from '@/components/now.js';
import {SECOND} from '@/util/date.js';
import {computed, ref, toValue, watch} from 'vue';

/**
 * @param {function(): PromiseLike<unknown>} fn - the function that will be polled
 * @param {MaybeRefOrGetter<number>?} period - how often to poll
 * @return {{
 *   isPolling: import('vue').Ref<boolean>,
 *   lastPoll: import('vue').Ref<Date|null>,
 *   nextPoll: import('vue').Ref<Date>,
 *   now: import('vue').Ref<Date>,
 *   shouldPoll: import('vue').Ref<boolean>,
 *   pollNow: (force?: boolean) => {}
 * }}
 */
export function usePoll(fn, period = 30 * SECOND) {
  const {now} = useNow(() => Math.min(SECOND, toValue(period)));
  const lastPoll = ref(/** @type {Date | null} */ null);
  const nextPoll = computed(() => new Date(lastPoll.value?.getTime() + toValue(period)));
  const isPolling = ref(false);
  const triggerPoll = ref(0);
  const shouldPoll = computed(() => {
    if (isPolling.value) return false;
    if (lastPoll.value === null) return true; // poll if we've never polled before
    if (triggerPoll.value > 0) return true; // explicitly asked to poll
    return (now.value >= nextPoll.value);
  });

  // Trigger polling now without waiting for the poll period to elapse
  const pollNow = (force = false) => {
    if (isPolling.value && force) {
      triggerPoll.value = 2;
    } else {
      triggerPoll.value = 1;
    }
  };

  watch(shouldPoll, async () => {
    while (shouldPoll.value) {
      isPolling.value = true;
      try {
        await fn();
      } catch (e) {
        // errors aren't important, the tracker will capture them
        console.debug('polling error:', e.message ?? e);
      } finally {
        lastPoll.value = new Date();
        isPolling.value = false;
        // force the poll to run again if we were already running when pollNow was called.
        triggerPoll.value = Math.max(0, triggerPoll.value - 1);
      }
    }
  }, {immediate: true});

  return {
    now,
    lastPoll,
    nextPoll,
    isPolling,
    shouldPoll,
    pollNow
  };
}
