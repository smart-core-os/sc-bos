import {timestampToDate} from '@/api/convpb.js';
import {listAllocationHistory} from '@/api/sc/traits/allocation.js';
import {asyncWatch} from '@/util/vue.js';
import {Allocation} from '@smart-core-os/sc-bos-ui-gen/proto/allocation_pb';
import binarySearch from 'binary-search';
import {computed, reactive, toValue} from 'vue';


/**
 * Returns the usage count for periods between edges.
 * The usage at index i will be the maximum usage for the period between edges[i] and edges[i+1].
 *
 * @param {import('vue').MaybeRefOrGetter<string[]>} names
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges
 * @return {import('vue').ComputedRef<{x: Date, groups: Record<string, {x: Date, y: number, last: number}>}[]>}
 */
export function useUsageCount(names, edges) {
  // As there's no way to MAX aggregate results by groupId from the history api we have to do it ourselves.
  // There could be quite a lot of data, we need to be careful not to keep it all around in memory.
  const countsByEdge = reactive(
      /** @type {{number: Record<string, {x: Date, y: number, last: number}>}} */
      {} // keyed by the leading edges .getTime()
  );
  asyncWatch([() => toValue(names), () => toValue(edges)], async ([names, edges], [oldNames, oldEdges]) => {
    if (names !== oldNames || edges !== oldEdges) {
      // if the names have changed, recompute all counts
      // likewise, if the edges have changed, counts from previous edges [and alternative time resolutions] are undesirable too.
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
      if (countsByEdge[edges[i].getTime()]) {
        continue;   // skip if already fetched data
      }

      const toFetch = [edges[i]];
      for (let j = i + 1; j < edges.length; j++) {
        const e = edges[j];
        toFetch.push(e);
        if (countsByEdge[e.getTime()]) {
          i = j - 1; // will i++ in the loop
          break;
        }
      }
      let countBefore = i > 0 ? countsByEdge[edges[i - 1]?.getTime()]?.last ?? null : null;
      const records = await readUsageCountSeries(names, toFetch, countBefore);
      if (Object.keys(records).length === 0) {
        countsByEdge[edges[i].getTime()] = {};
        continue;
      }
      for (const dataset in records) {
        const groupData = records[dataset];
        for (const record of groupData) {
          if (!countsByEdge[record.x.getTime()]) {
            countsByEdge[record.x.getTime()] = {};
          }

          countsByEdge[record.x.getTime()][dataset] = record.y;
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
 * Reads and calculates the maximum usage count for each span between the edges, returning it as a data series.
 * The y property at index i in the response will be the maximum usage count for the period between edges[i] and edges[i+1].
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

  const calcY = (d, beforeIdx) => {
    d[beforeIdx].last = Math.max(d[beforeIdx].last, d[beforeIdx].total, d[beforeIdx].sum);
    d[beforeIdx].y = Math.max(d[beforeIdx].total, d[beforeIdx].sum);
    if (d[beforeIdx].y < 0) d[beforeIdx].y = 0;
  }

  /** @type {({x: Date, y: number, total: number, sum: number, last: number}[])} */
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
      } catch (e) {
        console.error('Error reading allocation history for', name, e);
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

        if (!d) {
          d = [...dstArr]
        }

        if (d[beforeIdx]) {
          d[beforeIdx].total = Math.max((record.allocation.allocationTotal ?? 0) - (record.allocation.unallocationTotal ?? 0), d[beforeIdx].total);
          if (record.allocation.state === Allocation.State.ALLOCATED) {
            d[beforeIdx].sum += 1;
          } else if (record.allocation.state === Allocation.State.UNALLOCATED) {
            d[beforeIdx].sum -= 1;
          }
        } else {
          d[beforeIdx] = {
            x: before,
            sum: record.allocation.state === Allocation.State.ALLOCATED ? 1 : 0,
            total: (record.allocation.allocationTotal ?? 0) - (record.allocation.unallocationTotal ?? 0),
            last: 0
          };
        }

        calcY(d, beforeIdx);
        dst[record.allocation.groupId] = d;
      }

      req.pageToken = resp.nextPageToken;
    } while (req.pageToken);

    // handle edge pairs that don't have any records in them.
    // Usage count in subsequent spans will be the same as the last reading in the span before.
    // If the first span has no records, then we find the most recent record before the first edge.
    if (!countBefore) {
      copySrc = {x: edges[0], y: 0, total: 0, sum: 0, last: countBefore};
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
          const totalAssignedCount = rec.allocation.allocationTotal;
          if (totalAssignedCount > 1) {
            copySrc = {x: edges[0], y: totalAssignedCount, last: totalAssignedCount};
          } else {
            const sumAssignedCount = rec.allocation.state === Allocation.State.ALLOCATED ? 1 : 0
            copySrc = {x: edges[0], y: sumAssignedCount, last: sumAssignedCount};
          }
        }
      } catch {
        // ignore
      }
    }
  }

  for (const dataset in dst) {
    const d = dst[dataset];
    if (!d[0]) {
      d[0] = copySrc;
    }
    // if d[0] is still null, fill it with a null chart record so the subsequent fill works
    if (!d[0]) {
      d[0] = {x: edges[0], y: 0, last: 0};
    }
    // fill any null dst indexes with the value from the previous index
    for (let i = 1; i < d.length; i++) {
      if (!d[i]) {
        const last = d[i - 1].last;
        d[i] = {x: edges[i], y: 0, last: last};
      }
    }

    calcY(d, 0);
    dst[dataset] = d;
  }

  return dst;
}