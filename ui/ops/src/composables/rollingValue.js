import {computed, ref, toValue, watch} from 'vue';

/**
 * Tracks values over time to compute change metrics against a stable baseline.
 * By default, the baseline is the first non-null value received.
 * An optional baseline parameter can provide a fixed or historical reference.
 * Also tracks the previous sample's value for step-by-step comparisons.
 * More lightweight than rollingHistory - only stores necessary state.
 *
 * @param {import('vue').MaybeRefOrGetter<number|null>} value - The current value to track
 * @param {import('vue').MaybeRefOrGetter<number|null>} [baseline] - Optional fixed or historical baseline to override captured value
 * @return {{
 *   previousValue: import('vue').ComputedRef<number|null>,
 *   baselineValue: import('vue').ComputedRef<number|null>,
 *   percentChange: import('vue').ComputedRef<number|null>,
 *   absoluteChange: import('vue').ComputedRef<number|null>,
 *   hasChange: import('vue').ComputedRef<boolean>,
 *   reset: function():void
 * }} Previous value and change calculations
 */
export function useRollingValue(value, baseline = null) {
  const capturedBaseline = ref(null);
  const lastValue = ref(null);

  watch(() => toValue(value), (newValue, oldValue) => {
    // Skip null/undefined values
    if (newValue === null || newValue === undefined) return;

    // Store first valid value as baseline
    if (capturedBaseline.value === null) {
      capturedBaseline.value = newValue;
    }
    // Track previous sample's value
    lastValue.value = oldValue === undefined ? null : oldValue;
  }, {immediate: true});

  const effectiveBaseline = computed(() => {
    const b = baseline !== null ? toValue(baseline) : null;
    // Fall back to captured baseline if the explicit one is missing or zero (to avoid division by zero)
    if (b === null || b === 0) return capturedBaseline.value;
    return b;
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
    previousValue: computed(() => lastValue.value),
    baselineValue: effectiveBaseline,
    percentChange,
    absoluteChange,
    hasChange,
    reset: () => { capturedBaseline.value = null; }
  };
}
