import {useMeterReadingsAt} from '@/traits/meter/meter.js';
import {subDays, subMonths, subWeeks} from 'date-fns';
import {computed, toValue} from 'vue';

/**
 * Computes consumption (y = diff between readings) for two independent time windows on the same
 * meter. Use this to compare "current period" against any "baseline period".
 *
 * Can be generalised to any widget that shows meter data: pass any currentEdges and baselineEdges.
 * For example:
 *   - 7-day comparison: baselineEdges = currentEdges.map(d => subDays(d, 7))
 *   - Month-over-month: baselineEdges = currentEdges.map(d => subMonths(d, 1))
 *   - Custom range: pass arbitrary baseline edges
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} currentEdges
 * @param {import('vue').MaybeRefOrGetter<Date[]>} baselineEdges - same length as currentEdges
 * @return {{
 *   currentConsumption: import('vue').ComputedRef<{x:Date, y:number|null}[]>,
 *   baselineConsumption: import('vue').ComputedRef<{x:Date, y:number|null}[]>,
 *   deviations: import('vue').ComputedRef<{x:Date, current:number|null, baseline:number|null, pct:number|null}[]>,
 *   summaryPct: import('vue').ComputedRef<number|null>
 * }}
 */
export function useComparedConsumption(name, currentEdges, baselineEdges) {
  const currentReadings = useMeterReadingsAt(name, currentEdges);
  const baselineReadings = useMeterReadingsAt(name, baselineEdges);

  const buildSeries = (readings, edges) =>
    computed(() => {
      const res = [];
      const _edges = toValue(edges);
      const _readings = toValue(readings);
      for (let i = 1; i < _edges.length; i++) {
        const start = _readings[i - 1];
        const end = _readings[i];
        if (!start || !end) {
          res.push({x: _edges[i - 1], y: null});
          continue;
        }
        res.push({x: _edges[i - 1], y: end.usage - start.usage});
      }
      return res;
    });

  const currentConsumption = buildSeries(currentReadings, currentEdges);
  const baselineConsumption = buildSeries(baselineReadings, baselineEdges);

  const deviations = computed(() => {
    const cur = currentConsumption.value;
    const bas = baselineConsumption.value;
    return cur.map((pt, i) => {
      const base = bas[i]?.y ?? null;
      const current = pt.y;
      const pct = current !== null && base !== null && base !== 0
        ? ((current - base) / base) * 100
        : null;
      return {x: pt.x, current, baseline: base, pct};
    });
  });

  // Overall % vs baseline across all buckets (total current vs total baseline)
  const summaryPct = computed(() => {
    const cur = currentConsumption.value.reduce((acc, pt) => acc + (pt.y ?? 0), 0);
    const bas = baselineConsumption.value.reduce((acc, pt) => acc + (pt.y ?? 0), 0);
    if (!bas) return null;
    return ((cur - bas) / bas) * 100;
  });

  return {currentConsumption, baselineConsumption, deviations, summaryPct};
}

/**
 * Convenience wrapper: compares the current period against the same period shifted back by N days.
 * The `shiftFn` parameter lets callers customise the shift (default: 7 days).
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @param {(date: Date) => Date} [shiftFn] - how to compute the baseline date from a current date
 * @return {ReturnType<typeof useComparedConsumption>}
 */
export function useEnergyNormalized(name, edges, shiftFn = (d) => subDays(d, 7)) {
  const baselineEdges = computed(() => toValue(edges).map(shiftFn));
  return useComparedConsumption(name, edges, baselineEdges);
}

/**
 * Shift helpers re-exported for convenience so callers don't need to import date-fns directly.
 */
export {subDays, subWeeks, subMonths};
