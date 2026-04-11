import {useMaxPeopleCount} from '@/dynamic/widgets/occupancy/occupancy.js';
import {subDays, subMonths, subWeeks} from 'date-fns';
import {computed, toValue} from 'vue';

/**
 * Computes average people count for two independent time windows.
 * Use this to compare "current period" against any "baseline period".
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} currentEdges
 * @param {import('vue').MaybeRefOrGetter<Date[]>} baselineEdges - same length as currentEdges
 * @return {{
 *   currentCounts: import('vue').ComputedRef<{x:Date, y:number|null}[]>,
 *   baselineCounts: import('vue').ComputedRef<{x:Date, y:number|null}[]>,
 *   summaryPct: import('vue').ComputedRef<number|null>
 * }}
 */
export function useComparedOccupancy(name, currentEdges, baselineEdges) {
  const currentCounts = useMaxPeopleCount(name, currentEdges);
  const baselineCounts = useMaxPeopleCount(name, baselineEdges);

  // Overall % vs baseline across all buckets (average current vs average baseline)
  const summaryPct = computed(() => {
    const curVals = currentCounts.value.filter(pt => pt?.y != null && !isNaN(pt.y)).map(pt => pt.y);
    const basVals = baselineCounts.value.filter(pt => pt?.y != null && !isNaN(pt.y)).map(pt => pt.y);

    if (curVals.length === 0 || basVals.length === 0) return null;

    const curAvg = curVals.reduce((acc, val) => acc + val, 0) / curVals.length;
    const basAvg = basVals.reduce((acc, val) => acc + val, 0) / basVals.length;

    if (basAvg === 0 || isNaN(basAvg) || isNaN(curAvg)) return null;
    return ((curAvg - basAvg) / basAvg) * 100;
  });

  return {currentCounts, baselineCounts, summaryPct};
}

/**
 * Convenience wrapper: compares the current period against the same period shifted back.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @param {import('vue').MaybeRefOrGetter<(date: Date) => Date>} [shiftFn] - how to compute the baseline date from a current date
 * @return {ReturnType<typeof useComparedOccupancy>}
 */
export function useOccupancyNormalized(name, edges, shiftFn = (d) => subWeeks(d, 1)) {
  const baselineEdges = computed(() => toValue(edges).map(toValue(shiftFn)));
  return useComparedOccupancy(name, edges, baselineEdges);
}

/**
 * Converts a baseline shift string to a date-shifting function.
 *
 * @param {'day'|'week'|'month'|string} str
 * @return {(d: Date) => Date}
 */
export function shiftFnFromStr(str) {
  switch (str) {
    case 'month': return (d) => subMonths(d, 1);
    case 'day': return (d) => subDays(d, 7);
    case 'week':
    default: return (d) => subWeeks(d, 1);
  }
}

/**
 * Shift helpers re-exported for convenience.
 */
export {subDays, subWeeks, subMonths};

