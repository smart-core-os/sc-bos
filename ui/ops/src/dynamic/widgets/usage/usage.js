import {timestampToDate} from '@/api/convpb.js';
import {listAllocationsHistory} from '@/api/sc/traits/allocation.js';
import {asyncWatch} from '@/util/vue.js';
import {Allocation} from '@smart-core-os/sc-bos-ui-gen/proto/allocation_pb.d.ts';
import binarySearch from 'binary-search';
import {computed, reactive, toValue} from 'vue';


/**
 * Returns the usage count for periods between edges.
 * The usage at index i will be the total usage for the period between edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x:Date, y:number|null}[]>}
 */
export function useUsageCount(name, edges) {
  // As there's no way to aggregate results from the history api we have to do it ourselves.
  // There could be quite a lot of data, we need to be careful not to keep it all around in memory.
  // Because we only care about the max, we can process each pair of edges and just remember the max.

  const countsByEdge = reactive(
      /** @type {Record<number, UsageCountRecord>} */
      {} // keyed by the leading edges .getTime()
  );
  asyncWatch([() => toValue(name), () => toValue(edges)], async ([name, edges], [oldName]) => {
    if (name !== oldName) {
      Object.keys(countsByEdge).forEach(k => delete countsByEdge[k]);
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
        const records = await readUsageCountSeries(name, toFetch, countBefore);
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
 * Reads and calculates the total usage count for each span between the edges, returning it as a data series.
 * The y property at index i in the response will be the total usage count for the period between edges[i] and edges[i+1].
 * The last property at index i will be the most recent recorded usage count for the period between edges[i] and edges[i+1].
 *
 * @param {string} name
 * @param {Date[]} edges
 * @param {number | null} [countBefore] - the usage count before the first edge, if not null
 * @return {Promise<{x: Date, y: number, last: number}[]>} - of size edges.length - 1
 */
async function readUsageCountSeries(name, edges, countBefore) {
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

  /** @type {({x: Date, y: number, last: number})[]} */
  const dst = Array(edges.length - 1).fill(null);

  const req = {
    name,
    period: {
      startTime: edges[0],
      endTime: edges[edges.length - 1],
    },
    pageSize: 1000,
    orderBy: 'record_time DESC'
  };

  do {
    const resp = await listAllocationsHistory(req);

    if (resp.allocationsList.length === 0) break;

    let {before, after, beforeIdx} = findEdges(edges, timestampToDate(resp.allocationsList[0].recordTime));

    for (const record of resp.allocationsList) {
      const recordTime = timestampToDate(record.recordTime);
      if (recordTime < before || recordTime >= after) {
        ({before, after, beforeIdx} = findEdges(edges, recordTime));
        if (!after) break;
      }
      const old = dst[beforeIdx];
      if (!old || record.allocation.assignment > Allocation.Assignment.UNASSIGNED) {
        if (dst[beforeIdx])
          dst[beforeIdx]['y']++;
        else
          dst[beforeIdx] = {x: before, y: 1, last: 0};
      }

      dst[beforeIdx].last = record.allocation.assignment > Allocation.Assignment.UNASSIGNED ? dst[beforeIdx].last + 1 : dst[beforeIdx].last;
    }

    req.pageToken = resp.nextPageToken;
  } while (req.pageToken);


  // handle edge pairs that don't have any records in them.
  // Usage count in subsequent spans will be the same as the last reading in the span before.
  // If the first span has no records, then we find the most recent record before the first edge.
  if (dst[0] === null) {
    if (countBefore !== null) {
      dst[0] = {x: edges[0], y: countBefore, last: countBefore};
    } else {
      try {
        const res = await listAllocationsHistory({
          name,
          period: {endTime: edges[0]},
          orderBy: 'record_time desc',
          pageSize: 1,
        });
        if (res.allocationsList.length > 0) {
          const rec = res.allocationsList[0];
          const assignedCount = rec.allocation.assignment > Allocation.Assignment.UNASSIGNED ? 1 : 0;
          dst[0] = {x: edges[0], y: assignedCount, last: assignedCount};
        }
      } catch (e) {
        console.error('failed to fetch usage count before first edge', edges[0], e);
      }
    }
  }

  return dst;
}