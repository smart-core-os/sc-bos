
import {HOUR, MINUTE, roundDown} from '@/util/date.js';
import {computed, toValue} from 'vue';

/**
 * @param {MaybeRefOrGetter<Date>}now
 * @param {MaybeRefOrGetter<number>} [span]
 * @param {MaybeRefOrGetter<number>} [size]
 * @return {{periodStart: import('vue').ComputedRef<Date>, periodEnd: import('vue').ComputedRef<Date>}}
 */
export default function(now, span = 15 * MINUTE, size = 24 * HOUR) {
  const nowMinusSize = computed(() => new Date(toValue(now).getTime() - toValue(size)));
  const periodStart = computed(() => roundDown(toValue(nowMinusSize), toValue(span)));
  const periodEnd = computed(() => roundDown(toValue(now), toValue(span)));

  return {
    periodStart,
    periodEnd
  };
}
