import {timestampToDate} from '@/api/convpb.js';
import {listAirQualitySensorHistory} from '@/api/sc/traits/air-quality-sensor.js';
import {asyncWatch} from '@/util/vue.js';
import binarySearch from 'binary-search';
import {computed, reactive, toValue, watch, effectScope} from 'vue';
import {usePullMetadata} from '@/traits/metadata/metadata.js';


/**
 * @typedef {Object} AirQualityMetricRecord
 * @property {Date} x - the date of the record
 * @property {number|null} y - the average value for the metric over the span
 * @property {number|null} last - the last recorded metric for the span
 */

/**
 * Returns the average air quality metric for periods between edges.
 * The metric at index i will be the average for the period between edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<keyof AirQuality.AsObject>} metric
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x:Date, y:number|null}[]>}
 */
export function useAirQualityHistoryMetric(name, metric, edges) {
  const recordsByEdge = reactive(
      /** @type {Record<number, AirQualityMetricRecord>} */
      {} // keyed by the leading edges .getTime()
  );
  asyncWatch([() => toValue(name), () => toValue(metric), () => toValue(edges)], async ([name, metric, edges], [oldName, oldMetric]) => {
    if (name !== oldName || metric !== oldMetric) {
      Object.keys(recordsByEdge).forEach(k => delete recordsByEdge[k]);
    }

    if (edges.length < 2) {
      console.warn('useAirQualityHistoryMetric: edges must have at least 2 elements', edges);
      return;
    }

    const toDelete = new Set(Object.keys(recordsByEdge));
    for (const edge of edges) {
      toDelete.delete(edge.getTime().toString());
    }
    for (const k of toDelete) {
      delete recordsByEdge[k];
    }

    for (let i = 0; i < edges.length - 1; i++) {
      const leadingEdge = edges[i];
      if (!recordsByEdge[leadingEdge.getTime()]) {
        const toFetch = [leadingEdge];
        for (let j = i + 1; j < edges.length; j++) {
          const e = edges[j];
          toFetch.push(e);
          if (recordsByEdge[e.getTime()]) {
            i = j - 1; // will i++ in the loop
            break;
          }
        }

        const valBefore = recordsByEdge[edges[i-1]?.getTime()]?.last ?? null;
        const records = await readAverageAirQualityMetricSeries(name, metric, toFetch, valBefore);
        for (const record of records) {
          recordsByEdge[record.x.getTime()] = record;
        }
      }
    }
  }, {immediate: true});
  return computed(() => {
    const values = Object.values(recordsByEdge);
    values.sort((a, b) => a.x.getTime() - b.x.getTime());
    return values;
  });
}

/**
 * Reads historical air quality data for each of the edges and averages the specified metric returning as a chart data series.
 * The y property at index i in the response will be the average air quality metric for the period between edges[i] and edges[i+1].
 * The last property at index i will be the most recent recorded air quality metric and the record time for the period between edges[i] and edges[i+1].
 *
 * @param {string} name
 * @param {keyof AirQuality.AsObject} metric
 * @param {Date[]} edges
 * @param {number | null} [before]
 * @return {Promise<AirQualityMetricRecord[]>}
 */
async function readAverageAirQualityMetricSeries(name, metric, edges, before = null) {
  const findEdges = (edges, at) => {
    let i = binarySearch(edges, at, (a, b) => a.getTime() - b.getTime());
    if (i < 0) {
      // binarySearch will return the index _after_ the edge before the span as this is where the value would be inserted
      i = ~i - 1;
    }
    const res = {beforeIdx: i, before: edges[i], after: null, afterIdx: null};
    if (i < edges.length - 1) {
      res.after = edges[i + 1];
      res.afterIdx = i + 1;
    }
    return res;
  }

  /**
   * @typedef {Object} AvgCollector
   * @property {number} sum
   * @property {number} count
   * @property {number|null} last
   */
  /**
   * @type {AvgCollector[]}
   */
  const spans = Array(edges.length - 1).fill(null)
      .map(() => ({sum: 0, count: 0, last: null}));

  const req = {
    name,
    period: {
      startTime: edges[0],
      endTime: edges[edges.length - 1],
    },
    pageSize: 500, // trade off between items in memory at once and number of requests
  };
  do {
    const res = await listAirQualitySensorHistory(req, {});
    if (res.airQualityRecordsList.length === 0) break; // just in case, no infinite loop
    let before, after, beforeIdx;
    for (const record of res.airQualityRecordsList) {
      const d = timestampToDate(record.recordTime);
      if (!before || d < before || d >= after) {
        ({before, after, beforeIdx} = findEdges(edges, d));
        if (!after) {
          break; // the server returned records outside our query
        }
      }
      const val = record.airQuality[metric];
      const span = spans[beforeIdx];
      span.sum += val;
      span.count++;
      span.last = val;
    }
    req.pageToken = res.nextPageToken;
  } while (req.pageToken);

  const dst = spans.map((span, i) => {
    const record = {x: edges[i], y: null, last: null};
    if (span.count > 0) {
      record.y = span.sum / span.count;
      record.last = span.last;
    }
    return record;
  });

  // handle edge pairs that don't have any records in them.
  // Air quality in subsequent spans will be the same as the last reading in the span before.
  // If the first span has no records, then we find the most recent record before the first edge.
  if (dst[0].y === null) {
    if (before !== null) {
      dst[0].y = before;
      dst[0].last = before;
    } else {
      try {
        const res = await listAirQualitySensorHistory({
          name,
          period: {endTime: edges[0]},
          orderBy: 'record_time desc',
          pageSize: 1,
        }, {});
        if (res.airQualityRecordsList.length > 0) {
          const record = res.airQualityRecordsList[0];
          const val = record.airQuality[metric];
          dst[0].y = val;
          dst[0].last = val;
        }
      } catch (e) {
        console.error('Error reading air quality history', e);
      }
    }
  }
  // fill any null values in the dst array with the last known value
  for (let i = 1; i < dst.length; i++) {
    if (dst[i].y === null) {
      const last = dst[i - 1].last;
      dst[i].y = last;
      dst[i].last = last;
    }
  }
  return dst; // dst should have no null entries by now
}

/**
 * @typedef {Object} AirQualityMetric
 * @property {import('vue').MaybeRefOrGetter<string>} title
 * @property {import('vue').ComputedRef<{x:Date, y:number|null}[]>} data
 * @property {function():void} stop
 */
/**
 * @typedef {Object} ConfigSubName
 * @property {string} name
 * @property {string} [title]
 */
/**
 * @param {import('vue').MaybeRefOrGetter<(string|ConfigSubName)[]>} names
 * @param {import('vue').MaybeRefOrGetter<string>} metric
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').Reactive<Record<string, AirQualityMetric>>}
 */
export function useAirQualityHistoryMetrics(names, metric, edges) {
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
        const airQuality = {data: useAirQualityHistoryMetric(name, metric, edges), stop: () => scope.stop()};
        // use the configured title if possible, otherwise get it from the metadata, or just fall back to the name
        if (title) {
          airQuality.title = title;
        } else {
          const {value: md} = usePullMetadata(name);
          airQuality.title = computed(() => {
            const mdTitle = md.value?.appearance?.title;
            if (mdTitle) return mdTitle;
            return name;
          })
        }

        res[name] = airQuality;
      });
    }

    for (const [name, {stop}] of Object.entries(toStop)) {
      stop();
      delete res[name];
    }
  }, {immediate: true});
  return res;
}
