import {timestampToDate} from '@/api/convpb.js';
import {listAllocationHistory} from '@/api/sc/traits/allocation.js';
import {asyncWatch} from '@/util/vue.js';
import binarySearch from 'binary-search';
import {computed, reactive, toValue} from 'vue';


/**
 * Returns the usage count for periods between edges.
 * The usage at index i will be the total usage for the period between edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string[]>} names
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x: Date, groups: Record<string, {x: Date, y: number, last: number}>}[]>}
 */
export function useUsageCount(names, edges) {
  // As there's no way to SUM aggregate results by groupId from the history api we have to do it ourselves.
  // There could be quite a lot of data, we need to be careful not to keep it all around in memory.

  const countsByEdge = reactive(
      /** @type {{number: Record<string, {x: Date, y: number, last: number}>}} */
      {} // keyed by the leading edges .getTime()
  );
  asyncWatch([() => toValue(names), () => toValue(edges)], async ([names, edges], [oldNames]) => {
    if (names !== oldNames) {
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

        const countBefore = countsByEdge[edges[i - 1]?.getTime()]?.last ?? null;
        const records = await readUsageCountSeries(names, toFetch, countBefore);
        for (const dataset in records) {
          const groupData = records[dataset];
          for (const record of groupData) {
            if (!countsByEdge[record.x.getTime()])
              countsByEdge[record.x.getTime()] = {};

            countsByEdge[record.x.getTime()][dataset] = record.y;
          }
        }
      }
    }
  }, {immediate: true});

  return computed(() => {
    return Object.entries(countsByEdge)
        .map(([timestamp, groupData]) => ({
          x: new Date(Number(timestamp)),
          groups: groupData
        }))
        .sort((a, b) => a.x.getTime() - b.x.getTime());
  });
}

/**
 * Reads and calculates the total usage count for each span between the edges, returning it as a data series.
 * The y property at index i in the response will be the total usage count for the period between edges[i] and edges[i+1].
 * The last property at index i will be the most recent recorded usage count for the period between edges[i] and edges[i+1].
 * The results are grouped by allocation group id.
 *
 * @param {string[]} names
 * @param {Date[]} edges
 * @param {number | null} [countBefore] - the usage count before the first edge, if not null
 * @return {Promise<{string: {x: Date, y: number, last: number}[]}>} - of size edges.length - 1
 */
async function readUsageCountSeries(names, edges, countBefore) {
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

  /** @type {({x: Date, y: number, last: number}[])} */
  const dstArr = Array(edges.length - 1).fill(null);
  /** @type {{string: {x: Date, y: number, last: number}[]}} */
  const dst = {};
  let copySrc;

  for (const name of names) {
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
      let resp;

      try {
        resp = await listAllocationHistory(req, {});
      } catch {
        break;
      }

      if (resp.allocationRecordsList.length === 0) break;

      let {before, after, beforeIdx} = findEdges(edges, timestampToDate(resp.allocationRecordsList[0].recordTime));

      for (const record of resp.allocationRecordsList) {
        let d = dst[record.allocation.groupId];
        const recordTime = timestampToDate(record.recordTime);
        if (recordTime < before || recordTime >= after) {
          ({before, after, beforeIdx} = findEdges(edges, recordTime));
          if (!after) break;
        }
          if (!d)
            d = [...dstArr]

          if (d[beforeIdx]) {
            d[beforeIdx].y+= record.allocation.allocationTotal;
          } else {
            d[beforeIdx] = {x: before, y: record.allocation.allocationTotal, last: 0};
          }

          d[beforeIdx].last = d[beforeIdx].last + record.allocation.allocationTotal;
          dst[record.allocation.groupId] = d;
      }

      req.pageToken = resp.nextPageToken;
    } while (req.pageToken);


    // handle edge pairs that don't have any records in them.
    // Usage count in subsequent spans will be the same as the last reading in the span before.
    // If the first span has no records, then we find the most recent record before the first edge.
    if (countBefore !== null) {
      copySrc = {x: edges[0], y: countBefore, last: countBefore};
    } else {
      try {
        const res = await listAllocationHistory({
          name,
          period: {endTime: edges[0]},
          orderBy: 'record_time desc',
          pageSize: 1,
        }, {});
        if (res.allocationRecordsList.length > 0) {
          const rec = res.allocationRecordsList[0];
          const assignedCount = rec.allocation.allocationTotal;
          copySrc = {x: edges[0], y: assignedCount, last: assignedCount};
        }
      } catch {
        // ignore
      }
    }
  }

  for (const dataset in dst) {
    const d = dst[dataset];
    if (d[0] === null) {
      d[0] = copySrc;
    }
    // if d[0] is still null, fill it with a null chart record so the subsequent fill works
    if (d[0] === null) {
      d[0] = {x: edges[0], y: 0, last: 0};
    }
    // fill any null dst indexes with the value from the previous index
    for (let i = 1; i < d.length; i++) {
      if (d[i] === null) {
        const last = d[i - 1].last;
        d[i] = {x: edges[i], y: 0, last: last};
      }
    }

    dst[dataset] = d;
  }

  return dst;
}