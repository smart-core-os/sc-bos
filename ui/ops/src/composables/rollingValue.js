import {computed, ref, toValue, watch} from 'vue';

/**
 * Tracks the first non-null value and computes percentage change from current to that baseline.
 * More lightweight than rollingHistory - only stores one baseline value.
 *
 * @param {import('vue').MaybeRefOrGetter<number|null>} value - The current value to track
 * @param {import('vue').MaybeRefOrGetter<number|null>} [baseline] - Optional fixed or historical baseline to override captured value
 * @return {{
 *   previousValue: import('vue').ComputedRef<number|null>,
 *   percentChange: import('vue').ComputedRef<number|null>,
 *   absoluteChange: import('vue').ComputedRef<number|null>,
 *   hasChange: import('vue').ComputedRef<boolean>,
 *   reset: function():void
 * }} Previous value and change calculations
 */
export function useRollingValue(value, baseline = null) {
  const capturedBaseline = ref(null);

  watch(() => toValue(value), (newValue) => {
    // Skip null/undefined values
    if (newValue === null || newValue === undefined) return;

    // Store first valid value as baseline
    if (capturedBaseline.value === null) {
      capturedBaseline.value = newValue;
    }
  }, {immediate: true});

  const effectiveBaseline = computed(() => {
    if (baseline !== null) return toValue(baseline);
    return capturedBaseline.value;
  });

  const absoluteChange = computed(() => {
    const current = toValue(value);
    const b = effectiveBaseline.value;

    if (current === null || current === undefined || b === null || b === undefined) {
      return null;
    }

    return current - b;
  });

  const percentChange = computed(() => {
    const current = toValue(value);
    const b = effectiveBaseline.value;

    if (current === null || current === undefined || b === null || b === undefined) {
      return null;
    }

    // Avoid division by zero
    if (b === 0) return null;

    return ((current - b) / Math.abs(b)) * 100;
  });

  const hasChange = computed(() => {
    return absoluteChange.value !== null && Math.abs(absoluteChange.value) > 0.001;
  });

  return {
    previousValue: effectiveBaseline,
    percentChange,
    absoluteChange,
    hasChange,
    reset: () => { capturedBaseline.value = null; }
  };
}
