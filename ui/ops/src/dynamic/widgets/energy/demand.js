import {timestampToDate} from '@/api/convpb.js';
import {getDemand, listElectricDemandHistory} from '@/api/sc/traits/electric.js';
import {useAction} from '@/composables/action.js';
import {usePullMetadata} from '@/traits/metadata/metadata.js';
import {computed, effectScope, reactive, ref, toValue, watch} from 'vue';

/**
 * Returns the first property name in priorities that exists (and is non-zero) from the first non-null name.
 *
 * @param {import('vue').MaybeRefOrGetter<string[]>} priorities
 * @param {import('vue').MaybeRefOrGetter<string[]>} names
 * @return {import('vue').ComputedRef<string|null>}
 */
export function usePresentMetric(priorities, names) {
  const firstName = computed(() => toValue(names).find(v => Boolean(v)));
  const request = computed(() => {
    const props = toValue(priorities);
    if (props.length <= 1) return null; // no need to fetch if only one property
    const name = firstName.value;
    if (!name) return null;
    return {name}
  })
  const {response} = useAction(request, getDemand)
  return computed(() => {
    const props = toValue(priorities);
    if (props.length === 0) return null;
    if (props.length === 1) return props[0]; // only one property, always return it

    const demand = response.value;
    if (!demand) return null;
    for (const p of props) {
      if (Object.hasOwn(demand, p) && demand[p] && demand[p] !== 0) {
        return p;
      }
    }
    return null
  })
}

export const Units = {
  'current': 'A',
  'realPower': 'kW',
  'apparentPower': 'kVA',
  'reactivePower': 'kVAR',
  'powerFactor': '',
  'voltage': 'V',
}
const div = (v, fac) => {
  if (v === null || v === undefined) return null;
  return v / fac;
}
const selectors = {
  'current': (record) => record?.current,
  'realPower': (record) => div(record?.realPower, 1000),
  'apparentPower': (record) => div(record?.apparentPower, 1000),
  'reactivePower': (record) => div(record?.reactivePower, 1000),
  'powerFactor': (record) => record?.powerFactor,
  'voltage': (record) => record?.voltage,
}

/**
 * Returns a summary of the demand for each pair of edges using the given metric.
 * The summary calculation may change depending on the metric being used,
 * though most tend to use and average of records between the edges.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @param {import('vue').MaybeRefOrGetter<string>} metric
 * @return {import('vue').ComputedRef<{x:Date, y:number|null}[]>}
 */
export function useDemand(name, edges, metric) {
  const select = computed(() => {
    const m = toValue(metric);
    if (!m) return null;
    return selectors[m] || null;
  });
  const aggregate = (values) => {
    if (values.length === 0) return null;
    const sum = values.reduce((a, b) => a + b, 0);
    return sum / values.length;
  }
  const between = useDemandBetween(name, edges, select, aggregate);
  return computed(() => {
    const _edges = toValue(edges);
    const _between = toValue(between);
    const res = [];
    for (let i = 0; i < _between.length; i++) {
      res.push({x: _edges[i], y: _between[i]});
    }
    return res;
  });
}

/**
 * @typedef {Object} Series
 * @property {import('vue').MaybeRefOrGetter<string>} title
 * @property {import('vue').ComputedRef<{x:Date, y:number|null}[]>} data
 * @property {function():void} stop
 */
/**
 * @typedef {Object} SeriesName
 * @property {string} name
 * @property {string} [title]
 */
/**
 * Returns a reactive object mapping each name to its demand series.
 * Series are added and removed as names are added and removed from the input.
 * Each series contains a title (either from the config, metadata, or the name itself).
 *
 * @param {import('vue').MaybeRefOrGetter<(string|SeriesName)[]>} names
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @param {import('vue').MaybeRefOrGetter<string>} metric
 * @return {import('vue').Reactive<Record<string, Series>>}
 */
export function useDemands(names, edges, metric) {
  const res = reactive({});
  watch(() => toValue(names), (names) => {
    const toStop = Object.fromEntries(Object.entries(res)); // clone
    for (const item of names) {
      let name = item;
      let title = undefined;
      if (typeof name === 'object') {
        name = item.name;
        title = item.title;
      }
      if (res[name]) {
        delete toStop[name];
        continue;
      }
      const scope = effectScope();
      scope.run(() => {
        const series = {data: useDemand(name, edges, metric), stop: () => scope.stop()};
        // use the configured title if possible, otherwise get it from the metadata, or just fall back to the name
        if (title) {
          series.title = title;
        } else {
          const {value: md} = usePullMetadata(name);
          series.title = computed(() => {
            const mdTitle = md.value?.appearance?.title;
            if (mdTitle) return mdTitle;
            return name;
          })
        }

        res[name] = series;
      });
    }

    for (const [name, {stop}] of Object.entries(toStop)) {
      stop();
      delete res[name];
    }
  }, {immediate: true});
  return res;
}

/**
 * Returns an array of the results of applying aggregate to all demand records between each pair of edges.
 * The result at index i is the result of applying aggregate to all records between edges[i] and edges[i+1].
 * If there are no records between a pair of edges, the result for that pair is null.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @param {import('vue').MaybeRefOrGetter<null|function(import('@smart-core-os/sc-api-grpc-web/traits/electric_pb').ElectricDemand.AsObject):(number|null)>} select
 * @param {function(number[]):number} aggregate
 * @return {import('vue').ComputedRef<(number|null)[]>}
 */
function useDemandBetween(name, edges, select, aggregate) {
  // Our history query must real all records that lie between the earliest and latest edge.
  // We don't have to keep all these records in memory though.

  const baseQuery = computed(() => {
    const _edges = toValue(edges);
    if (_edges.length < 2) return null;
    const nameVal = toValue(name);
    if (!nameVal) return null;
    return {
      name: nameVal,
      period: {
        start_time: _edges[0],
        end_time: _edges[_edges.length - 1],
      },
      pageSize: 1000, // we're likely going to be getting a lot of records
      // we want to get the newest records first, this makes the most sense for charts
      orderBy: 'record_time DESC',
    }
  });

  /**
   * @typedef {Object} Segment
   * @property {number[]} entries
   */

  /**
   * Compacts segment entries into a single entry using the aggregate function.
   *
   * @param {Segment} segment
   */
  const compact = (segment) => {
    if (segment.entries.length <= 1) return;
    segment.entries = [aggregate(segment.entries)];
  }
  /** @type {import('vue').Reactive<Record<number, Segment>>} */
  const segments = reactive({}); // keyed by Date.getTime()

  /**
   * Updates segments with the data found in records.
   * Modified segments will be compacted.
   *
   * @param {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/electric/v1/electric_history_pb').ElectricDemandRecord.AsObject[]} records - in descending order
   * @param {Date[]} edges - in ascending order
   */
  const processPage = (records, edges) => {
    if (records.length === 0) return;
    if (edges.length < 2) return;
    const _select = toValue(select);

    let edgeIndex = edges.length - 2; // start at the end
    let edgeTime = edges[edgeIndex].getTime();
    for (const record of records) {
      const recordTime = timestampToDate(record.recordTime).getTime();
      for (; edgeIndex >= 0 && recordTime < edgeTime; edgeIndex--) {
        // move to the next edge
        edgeTime = edges[edgeIndex].getTime();
      }
      if (edgeIndex < 0) break; // no more edges to process

      // record is between edges[edgeIndex] and edges[edgeIndex+1]
      const segmentKey = edgeTime;
      const segment = segments[segmentKey] ??= {entries: []};
      const value = _select?.(record.electricDemand);
      if (value !== null && value !== undefined) {
        segment.entries.push(value);
      }
    }

    // compact all segments
    for (const segment of Object.values(segments)) {
      compact(segment);
    }
  }

  // abort is used to exit early from the following action,
  // which would block until all pages have been read.
  const abort = ref(false);
  watch(baseQuery, () => abort.value = true);

  useAction(baseQuery, async (req) => {
    const _edges = toValue(edges);
    const pageReq = {...req, pageToken: ''};
    try {
      do {
        const res = await listElectricDemandHistory(pageReq)
        processPage(res.electricDemandRecordsList, _edges);
        pageReq.pageToken = res.nextPageToken;
      } while (pageReq.pageToken !== '' && !abort.value);
    } catch (e) {
      console.error('failed to list electric demand history for', req.name, ':', e.message ?? e);
    } finally {
      abort.value = false;
    }
  })

  return computed(() => {
    const _edges = toValue(edges);
    if (_edges.length < 2) return [];
    return _edges.slice(0, -1).map(e => {
      const segment = segments[e.getTime()];
      if (!segment || segment.entries.length === 0) return null;
      return segment.entries[0];
    });
  })
}