import {computed, ref, toValue, watch} from 'vue';

/**
 * Tracks the first non-null value and computes percentage change from current to that baseline.
 * More lightweight than rollingHistory - only stores one baseline value.
 *
 * @param {import('vue').MaybeRefOrGetter<number|null>} value - The current value to track
 * @return {{
 *   previousValue: import('vue').ComputedRef<number|null>,
 *   percentChange: import('vue').ComputedRef<number|null>,
 *   absoluteChange: import('vue').ComputedRef<number|null>,
 *   hasChange: import('vue').ComputedRef<boolean>
 * }} Previous value and change calculations
 */
export function useRollingValue(value) {
  const baselineValue = ref(null);

  watch(() => toValue(value), (newValue) => {
    // Skip null/undefined values
    if (newValue === null || newValue === undefined) return;

    // Store first valid value as baseline
    if (baselineValue.value === null) {
      baselineValue.value = newValue;
    }
  }, {immediate: true});

  const absoluteChange = computed(() => {
    const current = toValue(value);
    const baseline = baselineValue.value;

    if (current === null || current === undefined || baseline === null || baseline === undefined) {
      return null;
    }

    return current - baseline;
  });

  const percentChange = computed(() => {
    const current = toValue(value);
    const baseline = baselineValue.value;

    if (current === null || current === undefined || baseline === null || baseline === undefined) {
      return null;
    }

    // Avoid division by zero
    if (baseline === 0) return null;

    return ((current - baseline) / Math.abs(baseline)) * 100;
  });

  const hasChange = computed(() => {
    return absoluteChange.value !== null && Math.abs(absoluteChange.value) > 0.001;
  });

  return {
    previousValue: computed(() => baselineValue.value),
    percentChange,
    absoluteChange,
    hasChange,
    reset: () => { baselineValue.value = null; }
  };
}
