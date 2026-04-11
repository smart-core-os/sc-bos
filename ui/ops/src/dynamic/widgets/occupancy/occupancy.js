import {timestampToDate} from '@/api/convpb.js';
import {listOccupancySensorHistory} from '@/api/sc/traits/occupancy.js';
import {asyncWatch} from '@/util/vue.js';
import binarySearch from 'binary-search';
import {sentenceCase} from 'change-case';
import {computed, reactive, toValue} from 'vue';

// Default colours/icons for thresholds that don't specify one (ascending occupancy = worsening state)
const _defaultThresholdColors = ['success-lighten-1', 'info', 'warning', 'error-lighten-1'];
const _defaultThresholdIcons = ['mdi-account-group-outline', 'mdi-account-group', 'mdi-account-multiple-plus', 'mdi-account-alert'];

/**
 * Resolves the active threshold entry for a given occupancy percentage.
 *
 * @param {number} percentage
 * @param {{percentage: number, str?: string, color?: string, icon?: string}[]|null} thresholds
 * @return {{label: string, color: string, icon: string}|null}
 */
export function resolveOccupancyThreshold(percentage, thresholds) {
  if (!thresholds || thresholds.length === 0) return null;
  for (let i = 0; i < thresholds.length; i++) {
    const t = thresholds[i];
    if (t.percentage >= percentage) {
      return {
        label: sentenceCase(t.str ?? ''),
        color: t.color ?? _defaultThresholdColors[i] ?? 'error-lighten-1',
        icon: t.icon ?? _defaultThresholdIcons[i] ?? 'mdi-account-alert',
      };
    }
  }
  // Percentage exceeds all thresholds — use the last one
  const lastIndex = thresholds.length - 1;
  const t = thresholds[lastIndex];
  return {
    label: sentenceCase(t.str ?? ''),
    color: t.color ?? _defaultThresholdColors[lastIndex] ?? 'error-lighten-1',
    icon: t.icon ?? _defaultThresholdIcons[lastIndex] ?? 'mdi-account-alert',
  };
}

/**
 * @typedef {Object} PeopleCountRecord
 * @property {Date} x - the date of the record
 * @property {number|null} y - the max people count for the span
 * @property {number|null} last - the last recorded people count for the span
 */

/**
 * Returns the maximum people count for periods between edges.
 * The occupancy at index i will be the maximum occupancy for the period between edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x:Date, y:number|null}[]>}
 */
export function useMaxPeopleCount(name, edges) {
  // As there's no way to aggregate results from the history api we have to do it ourselves.
  // There could be quite a lot of data, we need to be careful not to keep it all around in memory.
  // Because we only care about the max, we can process each pair of edges and just remember the max.

  const countsByEdge = reactive(
      /** @type {Record<number, PeopleCountRecord>} */
      {} // keyed by the leading edges .getTime()
  );
  asyncWatch([() => toValue(name), () => toValue(edges)], async ([name, edges], [oldName]) => {
    if (name !== oldName) {
      Object.keys(countsByEdge).forEach(k => delete countsByEdge[k]);
    }

    if (!name || edges.length < 2) {
      return;
    }

    const toDelete = new Set(Object.keys(countsByEdge));
    for (const edge of edges) {
      toDelete.delete(edge.getTime().toString());
    }
    for (const k of toDelete) {
      delete countsByEdge[k];
    }

    for (let i = 0; i < edges.length - 1; i++) {
      const leadingEdge = edges[i];
      if (!countsByEdge[leadingEdge.getTime()]) {
        const toFetch = [leadingEdge];
        for (let j = i + 1; j < edges.length; j++) {
          const e = edges[j];
          toFetch.push(e);
          if (countsByEdge[e.getTime()]) {
            i = j - 1; // will i++ in the loop
            break;
          }
        }

        const countBefore = countsByEdge[edges[i-1]?.getTime()]?.last ?? null;
        const records = await readMaxPeopleCountSeries(name, toFetch, countBefore);
        for (const record of records) {
          countsByEdge[record.x.getTime()] = record;
        }
      }
    }
  }, {immediate: true});

  return computed(() => {
    const values = Object.values(countsByEdge);
    values.sort((a, b) => a.x.getTime() - b.x.getTime());
    return values;
  });
}

/**
 * Reads and calculates the max people count for each span between the edges, returning it as a data series.
 * The y property at index i in the response will be the max people count for the period between edges[i] and edges[i+1].
 * The last property at index i will be the most recent recorded people count for the period between edges[i] and edges[i+1].
 *
 * @param {string} name
 * @param {Date[]} edges
 * @param {number | null} [countBefore] - the people count before the first edge, if not null
 * @return {Promise<PeopleCountRecord[]>} - of size edges.length - 1
 */
async function readMaxPeopleCountSeries(name, edges, countBefore = null) {
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

  /** @type {(PeopleCountRecord|null)[]} */
  const dst = Array(edges.length - 1).fill(null);

  const req = {
    name,
    period: {
      startTime: edges[0],
      endTime: edges[edges.length - 1],
    },
    pageSize: 500, // trade off between items in memory at once and number of requests
  }
  do {
    const res = await listOccupancySensorHistory(req, {});
    if (res.occupancyRecordsList.length === 0) break; // just in case, no infinite loop
    // these are the current edges we're working between, beforeIdx is the index in dst array
    let {before, after, beforeIdx} = findEdges(edges, timestampToDate(res.occupancyRecordsList[0].recordTime));
    for (const record of res.occupancyRecordsList) {
      const d = timestampToDate(record.recordTime);
      if (d < before || d >= after) {
        ({before, after, beforeIdx} = findEdges(edges, d));
        if (!after) {
          break; // the server returned records outside our query
        }
      }
      const old = dst[beforeIdx];
      if (!old || record.occupancy.peopleCount > old.y) {
        dst[beforeIdx] = {x: before, y: record.occupancy.peopleCount};
      }
      // last is used as the y value of the next slot if there are no records between those edges
      dst[beforeIdx].last = record.occupancy.peopleCount;
    }
    req.pageToken = res.nextPageToken;
  } while (req.pageToken)

  // handle edge pairs that don't have any records in them.
  // Occupancy in subsequent spans will be the same as the last reading in the span before.
  // If the first span has no records, then we find the most recent record before the first edge.
  if (dst[0] === null) {
    if (countBefore !== null) {
      dst[0] = {x: edges[0], y: countBefore, last: countBefore};
    } else {
      try {
        const res = await listOccupancySensorHistory({
          name,
          period: {endTime: edges[0]},
          orderBy: 'record_time desc',
          pageSize: 1,
        }, {});
        if (res.occupancyRecordsList.length > 0) {
          const count = res.occupancyRecordsList[0].occupancy.peopleCount;
          dst[0] = {x: edges[0], y: count, last: count};
        }
      } catch (e) {
        console.error('failed to fetch occupancy before first edge', edges[0], e)
      }
    }
  }
  // if dst[0] is still null, fill it with a null chart record so the subsequent fill works
  if (dst[0] === null) {
    dst[0] = {x: edges[0], y: null, last: null};
  }
  // fill any null dst indexes with the value from the previous index
  for (let i = 1; i < dst.length; i++) {
    if (dst[i] === null) {
      const last = dst[i - 1].last;
      dst[i] = {x: edges[i], y: last, last};
    }
  }

  return dst; // dst should have no null entries by now
}