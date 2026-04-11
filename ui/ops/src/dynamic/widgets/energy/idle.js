import {timestampToDate} from '@/api/convpb.js';
import {listOccupancySensorHistory, pullOccupancy} from '@/api/sc/traits/occupancy.js';
import {closeResource, newResourceValue} from '@/api/resource.js';
import {SECOND, useNow} from '@/components/now.js';
import {useMeterConsumption} from '@/dynamic/widgets/meter/consumption.js';
import {useMeterReadingAt} from '@/traits/meter/meter.js';
import {asyncWatch} from '@/util/vue.js';
import binarySearch from 'binary-search';
import {Occupancy} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/occupancysensor/v1/occupancy_sensor_pb';
import {computed, onScopeDispose, reactive, toValue, watch} from 'vue';

/**
 * Composable that tracks occupancy state changes for each time bucket defined by edges.
 * Returns all state changes within each bucket for time-weighted calculations.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name - The name of the occupancy sensor
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges - Array of Date objects defining bucket boundaries
 * @return {import('vue').ComputedRef<Array<Array<{timestamp: Date, state: Occupancy.State}>>>} Array of state changes per bucket
 */
function useOccupancyStateChangesPerBucket(name, edges) {
  const changesByEdge = reactive({});

  asyncWatch([() => toValue(name), () => toValue(edges)], async ([name, edges], [oldName]) => {
    if (name !== oldName) {
      Object.keys(changesByEdge).forEach(k => delete changesByEdge[k]);
    }
    if (!name || edges.length < 2) return;

    const findEdgeIdx = (edges, at) => {
      let i = binarySearch(edges, at, (a, b) => a.getTime() - b.getTime());
      if (i < 0) i = ~i - 1;
      return i;
    };

    // Initialize buckets with empty arrays
    const buckets = Array(edges.length - 1).fill(null).map(() => []);

    // Get state before first edge
    let stateBeforeFirst = Occupancy.State.STATE_UNSPECIFIED;
    try {
      const res = await listOccupancySensorHistory({
        name,
        period: {endTime: edges[0]},
        orderBy: 'record_time desc',
        pageSize: 1,
      }, {});
      if (res.occupancyRecordsList.length > 0) {
        stateBeforeFirst = res.occupancyRecordsList[0].occupancy.state;
      }
    } catch {
      // ignore
    }

    // Fetch all state changes in the period
    const req = {
      name,
      period: {startTime: edges[0], endTime: edges[edges.length - 1]},
      pageSize: 500,
    };

    try {
      do {
        const res = await listOccupancySensorHistory(req, {});
        if (res.occupancyRecordsList.length === 0) break;
        for (const record of res.occupancyRecordsList) {
          const d = timestampToDate(record.recordTime);
          const idx = findEdgeIdx(edges, d);
          if (idx < 0 || idx >= edges.length - 1) continue;
          buckets[idx].push({
            timestamp: d,
            state: record.occupancy.state
          });
        }
        req.pageToken = res.nextPageToken;
      } while (req.pageToken);
    } catch {
      // ignore
    }

    // Add initial state entry at the start of each bucket
    // This ensures we know the state at the beginning of each time period
    let lastKnownState = stateBeforeFirst;
    for (let i = 0; i < buckets.length; i++) {
      // Sort changes by timestamp to ensure correct order
      buckets[i].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
      
      // Add initial state at bucket start if not already present
      const bucketStart = edges[i].getTime();
      if (buckets[i].length === 0 || buckets[i][0].timestamp.getTime() > bucketStart) {
        buckets[i].unshift({timestamp: edges[i], state: lastKnownState});
      }
      
      // Update last known state for next bucket
      lastKnownState = buckets[i][buckets[i].length - 1].state;
    }

    for (let i = 0; i < edges.length - 1; i++) {
      changesByEdge[edges[i].getTime()] = buckets[i];
    }
  }, {immediate: true});

  return computed(() => {
    const _edges = toValue(edges);
    if (_edges.length < 2) return [];
    return _edges.slice(0, -1).map(e => changesByEdge[e.getTime()] ?? []);
  });
}

/**
 * Composable that calculates idle energy consumption by combining meter data with occupancy states.
 * Uses time-weighted calculation: if a bucket has multiple state changes, energy is proportionally
 * allocated based on time spent in each state.
 * For the current ongoing bucket, uses real-time meter reading to calculate current consumption.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} meterName - The name of the meter device
 * @param {import('vue').MaybeRefOrGetter<string>} occupancyName - The name of the occupancy sensor
 * @param {import('vue').MaybeRefOrGetter<Date[]>} edges - Array of Date objects for consumption buckets
 * @param {import('vue').MaybeRefOrGetter<Date[]>} pastEdges - Array of Date objects for occupancy state buckets
 * @param {import('vue').MaybeRefOrGetter<Occupancy.State|null>} [realtimeState] - Optional real-time occupancy state
 * @param {import('vue').MaybeRefOrGetter<{usage: number, produced?: number}|null>} [liveMeterReading] - Optional live meter readings
 * @param {import('vue').MaybeRefOrGetter<Array<{timestamp: Date, state: Occupancy.State}>>} [realtimeHistory] - Pull-stream state history
 * @param {import('vue').MaybeRefOrGetter<number>} [negligibleEnergy=0.001] - Values below this are treated as effectively zero
 * @return {{idleConsumption: ComputedRef<({x: *, y: null}|{x: *, y: null}|{x: *, y})[]>, totalConsumption: import('vue').ComputedRef<{x: Date, y: (number|null)}[]>, totalIdle: ComputedRef<{x: *, y: null}|{x: *, y: null}|{x: *, y}>, totalEnergy: ComputedRef<number>, wastePercent: ComputedRef<number|*>}} Object containing idle consumption metrics and percentages
 */
export function useIdleConsumption(meterName, occupancyName, edges, pastEdges, realtimeState, liveMeterReading, realtimeHistory, negligibleEnergy = 0.001) {
  const consumption = useMeterConsumption(meterName, edges);
  const stateChanges = useOccupancyStateChangesPerBucket(occupancyName, pastEdges);
  const {now} = useNow(SECOND); // Update every second for real-time idle time calculation

  // Find the bucket covering the current time for real-time overrides
  const ongoingBucketIdx = computed(() => {
    const e = toValue(edges);
    const nowMs = now.value.getTime();
    for (let i = 0; i < e.length - 1; i++) {
      if (e[i].getTime() <= nowMs && e[i + 1].getTime() > nowMs) {
        return i;
      }
    }
    return -1;
  });

  // When a live reading is supplied, also track the start of the current bucket for a real-time delta.
  // lastBucketStartTime returns null for past periods or when liveMeterReading is absent,
  // so useMeterReadingAt stays idle in those cases.
  const lastBucketStartTime = computed(() => {
    const idx = ongoingBucketIdx.value;
    if (idx === -1) return null;

    // Access liveMeterReading reactively within computed
    const liveReading = liveMeterReading ? toValue(liveMeterReading) : null;
    if (!liveReading) return null;
    const e = toValue(edges);
    return e[idx];
  });
  const lastBucketStartReading = useMeterReadingAt(meterName, lastBucketStartTime, true);

  const totalConsumption = computed(() => {
    const c = toValue(consumption);
    const liveVal = liveMeterReading ? toValue(liveMeterReading) : null;
    const startVal = lastBucketStartReading.value;
    const ongoingIdx = ongoingBucketIdx.value;

    return c.map((pt, i) => {
      const isOngoingBucket = i === ongoingIdx;

      // For the ongoing bucket, override energy with the pull-stream delta if available
      let bucketEnergy = pt.y;
      if (isOngoingBucket && liveVal?.usage != null && startVal?.usage != null) {
        bucketEnergy = Math.max(0, liveVal.usage - startVal.usage);
      }
      return {x: pt.x, y: bucketEnergy};
    });
  });

  const idleConsumption = computed(() => {
    const c = totalConsumption.value;
    const changes = toValue(stateChanges);
    const _edges = toValue(edges); // Use full edges for bucket boundaries
    const currentState = realtimeState ? toValue(realtimeState) : null;
    const history = realtimeHistory ? toValue(realtimeHistory) : [];
    const ongoingIdx = ongoingBucketIdx.value;
    const currentTime = now.value.getTime();

    return c.map((pt, i) => {
      const isOngoingBucket = i === ongoingIdx;
      const bucketEnergy = pt.y;

      // Check for null/undefined specifically, not falsy values (0 is valid!)
      if (bucketEnergy == null || !_edges[i] || !_edges[i + 1]) return {x: pt.x, y: null};

      const bucketStart = _edges[i].getTime();
      const bucketEnd = _edges[i + 1].getTime();
      const bucketDuration = bucketEnd - bucketStart;

      // Build allChanges array, adding real-time state for the last bucket
      const bucketChanges = changes[i] || [];
      const allChanges = [...bucketChanges];

      // For the ongoing bucket, build state timeline from pull-stream history.
      // This avoids incorrectly assuming the current state has been constant since bucket start.
      if (isOngoingBucket && allChanges.length === 0 && history.length > 0) {
        // Find the most recent observation at or before bucket start (baseline state)
        const beforeStart = [...history].reverse().find(e => e.timestamp.getTime() <= bucketStart);
        if (beforeStart) {
          allChanges.push({timestamp: new Date(bucketStart), state: beforeStart.state});
        }
        // Add any state transitions that occurred within this bucket
        for (const entry of history) {
          const t = entry.timestamp.getTime();
          if (t > bucketStart && t < currentTime) allChanges.push(entry);
        }
        // If nothing at/before bucket start, use the first observation within the bucket
        if (allChanges.length === 0) {
          const firstIn = history.find(e => e.timestamp.getTime() > bucketStart);
          if (firstIn) allChanges.push(firstIn);
        }
      }
      // Add a sentinel at the current time so the idle loop measures up to now
      if (isOngoingBucket && allChanges.length > 0 &&
          currentState && currentState !== Occupancy.State.STATE_UNSPECIFIED) {
        allChanges.push({timestamp: new Date(currentTime), state: currentState});
      }

      // Check if we have any state changes to work with (after adding real-time state)
      if (allChanges.length === 0) return {x: pt.x, y: null};

      // For the ongoing bucket use the first observation as the reference point so that
      // idleTime and elapsed share the same denominator and the proportion stays stable.
      const observationStart = isOngoingBucket && allChanges.length > 0
          ? Math.max(allChanges[0].timestamp.getTime(), bucketStart)
          : bucketStart;

      // Calculate time-weighted idle proportion
      let idleTime = 0;
      for (let j = 0; j < allChanges.length; j++) {
        const change = allChanges[j];
        const isIdle = change.state === Occupancy.State.UNOCCUPIED ||
                       change.state === Occupancy.State.IDLE;

        if (!isIdle) continue;

        // Calculate duration this state was active (clamp to observation window)
        const stateStart = Math.max(change.timestamp.getTime(), observationStart);
        // For the last state change in the ongoing bucket, use current time as end
        const stateEnd = j < allChanges.length - 1
            ? Math.min(allChanges[j + 1].timestamp.getTime(), bucketEnd)
            : (isOngoingBucket ? Math.min(currentTime, bucketEnd) : bucketEnd);

        idleTime += Math.max(0, stateEnd - stateStart);
      }

      const elapsed = isOngoingBucket
          ? Math.min(currentTime, bucketEnd) - observationStart
          : bucketDuration;
      const idleProportion = elapsed > 0 ? idleTime / elapsed : 0;

      return {
        x: pt.x,
        y: bucketEnergy * idleProportion,
      };
    });
  });

  const totalIdle = computed(() => {
    let sum = 0;
    let hasValue = false;
    for (const pt of idleConsumption.value) {
      if (pt.y != null) {
        sum += pt.y;
        hasValue = true;
      }
    }
    return hasValue ? sum : null;
  });

  const totalEnergy = computed(() => {
    let sum = 0;
    let hasValue = false;
    for (const pt of totalConsumption.value) {
      if (pt.y != null) {
        sum += pt.y;
        hasValue = true;
      }
    }
    return hasValue ? sum : null;
  });

  const historicalIdle = computed(() => {
    const ongoingIdx = ongoingBucketIdx.value;
    let sum = 0;
    let hasValue = false;
    for (let i = 0; i < idleConsumption.value.length; i++) {
      if (ongoingIdx !== -1 && i === ongoingIdx) continue;
      const pt = idleConsumption.value[i];
      if (pt.y != null) {
        sum += pt.y;
        hasValue = true;
      }
    }
    return hasValue ? sum : null;
  });

  const historicalEnergy = computed(() => {
    const ongoingIdx = ongoingBucketIdx.value;
    let sum = 0;
    let hasValue = false;
    for (let i = 0; i < totalConsumption.value.length; i++) {
      if (ongoingIdx !== -1 && i === ongoingIdx) continue;
      const pt = totalConsumption.value[i];
      if (pt.y != null) {
        sum += pt.y;
        hasValue = true;
      }
    }
    return hasValue ? sum : null;
  });

  const wastePercent = computed(() => {
    const total = totalEnergy.value;
    if (!total || total < toValue(negligibleEnergy)) return 0;
    return (totalIdle.value / total) * 100;
  });

  return {idleConsumption, totalConsumption, totalIdle, totalEnergy, historicalIdle, historicalEnergy, wastePercent};
}

/**
 * Composable that tracks current occupancy state in real-time using pull API.
 *
 * @param {import('vue').MaybeRefOrGetter<string>} name - The name of the occupancy sensor
 * @param {import('vue').MaybeRefOrGetter<number>} lookback - Lookback time in milliseconds
 * @return {{
 *   state: import('vue').ComputedRef<Occupancy.State>,
 *   stateHistory: import('vue').ComputedRef<Array<{timestamp: Date, state: Occupancy.State}>>
 * }} Current occupancy state and recent history
 */
export function usePullOccupancyState(name, lookback) {
  const resource = reactive(newResourceValue());
  onScopeDispose(() => closeResource(resource));

  // Track state history for the lookback period
  const stateHistory = reactive([]);

  watch([() => toValue(name), () => toValue(lookback)], ([name, lookback]) => {
    closeResource(resource);
    stateHistory.length = 0; // Clear history

    if (!name) return;

    pullOccupancy({name, updatesOnly: false}, resource);

    // Watch for new occupancy values and add to history
    watch(() => resource.value, (value) => {
      if (!value?.state) return;

      const t = value.stateChangeTime ? timestampToDate(value.stateChangeTime) : new Date();
      
      // Only add to history if it's a new state or a different transition time
      const isNew = stateHistory.length === 0 || 
                    stateHistory.some(h => h.timestamp.getTime() === t.getTime() && h.state === value.state) === false;

      if (isNew) {
        stateHistory.push({timestamp: t, state: value.state});
        stateHistory.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
      }

      // Clean up old entries beyond lookback period, but keep one as a baseline
      // so the state at the start of the lookback window is still known.
      const now = new Date();
      const cutoff = now.getTime() - lookback;
      while (stateHistory.length > 1 && stateHistory[1].timestamp.getTime() < cutoff) {
        stateHistory.shift();
      }
    }, {immediate: true});
  }, {immediate: true});

  const currentState = computed(() => resource.value?.state ?? Occupancy.State.STATE_UNSPECIFIED);

  return {
    state: currentState,
    stateHistory: computed(() => [...stateHistory])
  };
}
