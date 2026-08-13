<template>
  <div class="quality-wrapper">
    <div class="quality-header">
      <span class="quality-title">Meter data quality</span>
      <div class="header-right">
        <!-- Counted from the rows, not from the live probe alone. "7 of 7 OK"
             beside a dashboard showing no rating was the contradiction that
             prompted this: a meter can answer a live read and still be unusable
             for the rating period. -->
        <span
            v-if="health.total"
            class="summary"
            :class="health.ok === health.total ? 'summary--ok' : 'summary--warn'">
          {{ health.ok }} of {{ health.total }} usable
        </span>
        <span v-if="health.blocking" class="summary summary--err">
          {{ health.blocking }} blocking the rating
        </span>
        <v-btn
            size="x-small"
            variant="text"
            color="rgba(255,255,255,0.5)"
            :disabled="!rows.length"
            @click="exportCsv">
          Export CSV
        </v-btn>
        <v-btn
            size="x-small"
            variant="text"
            color="rgba(255,255,255,0.5)"
            @click="emit('refresh')">
          Refresh
        </v-btn>
      </div>
    </div>

    <div v-if="Object.keys(meterStatuses).length === 0" class="quality-empty">
      No meters configured. Add meter names to <code>nabersBaseBuilding.meterNames</code> in dashboards-config.json.
    </div>

    <div v-else class="quality-scroll">
      <table class="quality-table">
        <thead>
          <tr>
            <th>Ref</th>
            <th>Meter</th>
            <th>Category</th>
            <th>Status</th>
            <th>Current reading</th>
            <th>Resolution</th>
            <th>Longest gap in history</th>
            <th>Fault</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.name">
            <td class="ref-cell">{{ row.id.ref || '—' }}</td>
            <!-- The Smart Core name stays available on hover, for anyone who has
                 to go and interrogate the device, but it is no longer what the
                 table reads as. -->
            <td class="meter-name" :title="row.name">
              <span class="name-label">{{ row.id.label }}</span>
              <span v-if="row.id.location" class="name-location">{{ row.id.location }}</span>
            </td>
            <td>{{ categoryLabels[row.status.category] ?? row.status.category }}</td>
            <td>
              <span class="status-chip" :class="statusClass(row)" :title="statusTitle(row)">{{ statusLabel(row) }}</span>
              <!-- Discarding corrupt readings is a data-quality event in its own
                   right, so it is stated rather than done silently. -->
              <span v-if="row.rejected" class="discard-note" :title="discardTitle(row)">
                {{ row.rejected }} discarded
              </span>
            </td>
            <!-- The live reading stays in its own cell even when the meter is
                 unusable for the rating: "answers now, but cannot be rated" is
                 precisely the state that needs both facts side by side. -->
            <td class="reading-cell">
              {{ row.status.value !== null && row.status.ok
                ? row.status.value.toFixed(1) + ' kWh'
                : '—' }}
            </td>
            <td class="reading-cell tick-cell" :title="tickTitle(row)">{{ tickLabel(row) }}</td>
            <td class="reading-cell gap-cell" :title="gapTitle(row)">{{ row.gapLabel || '—' }}</td>
            <td class="fault-cell">{{ faultFor(row) || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!--
      Only what nothing else on screen says. Each status, the resolution and the
      discard note carry their own hover text, and the Fault column now prints
      the reason a meter is unusable, so all of that used to be told twice — at
      length. What is left is the handful of readings the table invites and gets
      wrong, plus the submission caveat, which has nowhere else to live.
    -->
    <div class="quality-desc">
      <p class="desc-hint">Hover any status, gap, resolution or discard note for its detail.</p>
      <p>
        <strong>Current reading</strong> is the meter's cumulative accumulator as it reads
        now, not consumption over the rating period.
      </p>
      <p class="usable-desc">
        <strong>No period data</strong> blanks the rating and the stat cards until it is
        resolved. Most often the meter's history begins after
        <code>ratingPeriodStart</code>, leaving no opening reading to measure from.
      </p>
      <p>
        All meters must read continuously for a valid NABERS submission, and non-utility
        meters require CT ratio validation by an accredited assessor.
      </p>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {format} from 'date-fns';
import {meterIdentity} from '@/stores/nabersBaseBuildingMetrics.js';
import {formatGap} from '@/util/meterEstimation.js';
import {downloadCsv} from '@/util/csv.js';

/**
 * The longest gap's duration and when it was — "66 days, 5 Oct–10 Dec".
 *
 * The dates are the point of it. A duration on its own cannot be acted on: an
 * engineer asked "when was this?" and the column had no answer, because the old
 * figure was a total across the period with no location at all.
 *
 * @param {{from: Date, to: Date, hours: number}|null} gap
 * @return {string}
 */
function gapLabelFor(gap) {
  if (!gap || !(gap.hours > 0)) return '';
  const span = formatGap(gap.hours);
  // Same year at both ends is the common case, so the year is stated once.
  const sameYear = gap.from.getFullYear() === gap.to.getFullYear();
  const from = format(gap.from, sameYear ? 'd MMM' : 'd MMM yy');
  const to = format(gap.to, 'd MMM yy');
  return `${span}, ${from}–${to}`;
}

const props = defineProps({
  meterStatuses:  {type: Object, default: () => ({})},
  categoryLabels: {type: Object, default: () => ({})},
  /**
   * `{[meterName]: {ref, label, floor, zone, location}}` from device metadata.
   * The EMS specification requires each meter to be presented by its
   * human-readable name and unique ref, not by its Smart Core path.
   */
  meterIdentities: {type: Object, default: () => ({})},
  /** `{ok, total}`, so the count is readable without expanding the panel. */
  /**
   * `{[meterName]: {estimatedHours, unrecordedHours, longestGap, …}}` over the
   * rating period. An assessor needs to know which board was estimated and for how
   * long, not merely that something was.
   *
   * `estimatedHours` is the substituted stretch, which now only ever comes from
   * projecting an unreachable meter forward. `unrecordedHours` is the stretch with
   * no records behind it, which is usually larger and is not a disclosure: history
   * records only changes, so a quiet accumulator held its last value and the figure
   * derived from it is exact.
   */
  meterEstimation: {type: Object, default: () => ({})},
  /**
   * Names of meters that answered a live read but have recorded nothing recent.
   *
   * `pkg/auto/history` records a meter reading only when it changes, so a present
   * meter whose accumulator has not moved writes nothing at all. Its figure is
   * exact and nothing was estimated — but a meter that ought to be consuming and
   * reads idle is a commissioning fault, so it earns its own state rather than
   * hiding under "OK".
   */
  meterIdle: {type: Array, default: () => []},
  /**
   * `{[meterName]: reason}` for meters whose period-to-date figure could not be
   * produced.
   *
   * Deliberately separate from `meterStatuses`, which is a live reachability
   * probe. The two answer different questions and can disagree: a meter that
   * answers right now may still have no history reaching back to the period
   * start, which blanks every figure on the dashboard. Without this the table
   * showed such a meter as OK with a current reading while the stat cards said
   * they were waiting on it.
   */
  meterFailureReasons: {type: Object, default: () => ({})}
});

const emit = defineEmits(['refresh']);

// The store builds `meterStatuses` from an ordered work list, so its key order is
// config order. Failures float to the top, then estimated meters: when a
// building lists every board individually the one that is down is what the
// reader came for, and the one that was projected is what they need next.
const rows = computed(() => {
  const idle = new Set(props.meterIdle);
  return Object.entries(props.meterStatuses)
    .map(([name, status], i) => {
      const q = props.meterEstimation[name] ?? {};
      const tick = q.observedTickKwh ?? null;
      const rejected = q.rejectedReadings ?? 0;
      // Two different figures. `hours` is what the reported total rests on and so
      // drives the status chip; `unrecorded` is how much of the period has no
      // history behind it, which is the gap column and can be much larger — a
      // meter can have months unrecorded and still yield an exact total, because
      // the total is just the difference between its two end readings.
      const hours = q.estimatedHours ?? 0;
      const unrecorded = q.unrecordedHours ?? 0;
      // `meterIdentity(name)` with no metadata still yields a usable fallback, so
      // a meter the store has not fetched yet renders rather than blanking.
      const id = props.meterIdentities[name] ?? meterIdentity(name);
      return {
        name, status, i, hours, unrecorded, tick, rejected,
        gap: q.longestGap ?? null, gapLabel: gapLabelFor(q.longestGap),
        // Why the rating cannot use this meter, which is not the same as whether
        // it answers a live read.
        periodFault: props.meterFailureReasons[name] ?? null,
        id, idle: idle.has(name)
      };
    })
    // Healthy first, worst last, then by how little history is missing. Config
    // order breaks the remaining ties and is deliberately *not* inverted: on a
    // building with nothing wrong every row ranks the same, and reversing that
    // would present the whole meter schedule backwards.
    .sort((a, b) =>
      (severity(a) - severity(b)) ||
      (a.unrecorded - b.unrecorded) ||
      (a.i - b.i));
});

/**
 * How much attention a row wants, lowest first.
 *
 * A rank rather than a chain of comparisons, so the order reads at a glance and
 * reversing it is one sign rather than five.
 *
 * @param {Object} row
 * @return {number}
 */
function severity(row) {
  if (!row.status.ok) return 4;      // does not answer at all
  if (row.periodFault) return 3;     // answers, but blocks the rating
  if (row.hours > 0) return 2;       // figure rests on a projected reading
  if (row.idle) return 1;            // present, consuming nothing
  return 0;
}

/**
 * Usable means the rating can actually use it: reachable *and* able to produce a
 * figure for the period. `blocking` is the count that cannot, each of which
 * blanks every dependent figure on the dashboard.
 */
const health = computed(() => {
  const all = rows.value;
  const blocking = all.filter(r => !r.status.ok || r.periodFault).length;
  return {ok: all.length - blocking, total: all.length, blocking};
});

/**
 * A meter can be reachable right now and still have had a gap earlier in the
 * period, which is exactly the case the rating cares about — so "OK" alone
 * would hide the thing being disclosed.
 *
 * `Idle` is deliberately not an error and not an estimate: history records only
 * changes, so a present meter with an unmoved accumulator writes nothing, and its
 * figure is exact.
 *
 * @param {Object} row
 * @return {string}
 */
function statusClass(row) {
  if (!row.status.ok) return 'status-err';
  if (row.periodFault) return 'status-err';
  if (row.hours > 0) return 'status-est';
  return row.idle ? 'status-idle' : 'status-ok';
}

/**
 * @param {Object} row
 * @return {string}
 */
function statusLabel(row) {
  if (!row.status.ok) return '✗ Error';
  // Reachable, but its history cannot produce a figure for the rating period, so
  // it blanks every dependent figure on the dashboard. Named for the consequence,
  // because "OK" here beside a dashboard showing no rating was the contradiction
  // that made this state worth having.
  if (row.periodFault) return '✗ No period data';
  if (row.hours > 0) return '~ Estimated';
  return row.idle ? '— Idle' : '✓ OK';
}

/**
 * @param {Object} row
 * @return {string|undefined}
 */
function statusTitle(row) {
  if (!row.status.ok) return row.status.error ?? undefined;
  if (row.periodFault) return row.periodFault;
  // Idle used to be explained in the panel's footnotes, where it read as a
  // defence of a state nobody had accused of anything. It belongs on the chip:
  // it is the only status whose meaning is genuinely counter-intuitive, and the
  // reader who needs it is the one looking straight at it.
  if (row.idle) {
    return 'Reachable, and exact — nothing here is estimated. Its accumulator has ' +
      'simply not moved beyond its own resolution for some time, and history records ' +
      'changes rather than ticks, so a meter consuming nothing writes nothing. ' +
      'Worth a look only if this board ought to be running: a live meter counting ' +
      'nothing is the one fault a silently-caching driver can hide.';
  }
  return undefined;
}

/**
 * The longest gap, and the thing about it that is read wrongly by default.
 *
 * A wide gap looks like missing data and is usually nothing of the kind, so the
 * column carries its own caveat rather than leaning on a footnote several
 * inches away from the figure that prompted the question.
 *
 * @param {Object} row
 * @return {string}
 */
function gapTitle(row) {
  if (!row.gap) return 'No stretch of the rating period is without recorded history.';
  return 'The widest single stretch of the rating period with no recorded history ' +
    'behind it, dated to the nearest month boundary. This does not mean the figures ' +
    'were estimated, and usually they were not: consumption is the difference between ' +
    'two readings, so a gap with real readings either side still totals exactly. Only ' +
    'the Estimated status means a value was substituted.';
}

/**
 * @param {Object} row
 * @return {string}
 */
function discardTitle(row) {
  return `${row.rejected} recorded reading(s) could not have come from a ` +
    'cumulative accumulator — negative, higher than the meter reads now, or a dip ' +
    'the very next reading recovered from — so they were excluded before any figure ' +
    'was derived. The figures are unaffected by the discard, but a board that ' +
    'discards readings repeatedly is a driver or comms fault worth chasing.';
}

/**
 * The meter's measured resolution — the smallest step it has been seen to take.
 *
 * `sub-kWh` rather than a string of decimals: the exact figure for a fine meter is
 * noise in a column read at a glance, and the hover carries it.
 *
 * @param {Object} row
 * @return {string}
 */
function tickLabel(row) {
  if (row.tick === null) return '—';
  if (row.tick < 0.5) return 'sub-kWh';
  return `${row.tick < 10 ? row.tick.toFixed(1) : Math.round(row.tick)} kWh`;
}

/**
 * @param {Object} row
 * @return {string|undefined}
 */
function tickTitle(row) {
  if (row.tick === null) {
    return 'Resolution not measurable from the readings held; the configured ' +
      'idleToleranceKwh is used instead.';
  }
  return `Smallest step observed: ${row.tick} kWh. Movement at or below this is ` +
    'treated as no consumption rather than as missing data.';
}

/**
 * Whichever fault explains this row, live first.
 *
 * An unreachable meter's live error is the more likely root cause, and its period
 * failure is usually a consequence of it, so showing both would be noise.
 *
 * @param {Object} row
 * @return {string}
 */
function faultFor(row) {
  if (!row.status.ok) return row.status.error ?? 'unreadable';
  return row.periodFault ?? '';
}

/**
 * Export the meter schedule.
 *
 * The monthly export is by month and end use, which is the right shape for a
 * rating but says nothing about which meter produced which figure. The EMS
 * specification wants every export to carry each meter's human-readable name, its
 * unique ref and its mapped reporting category, so this is the per-meter
 * dimension: one row per meter, with what it reads and how sound that reading is.
 */
function exportCsv() {
  const generatedAt = new Date();

  const preamble = [
    ['NABERS UK base building — meter schedule and data quality'],
    ['Generated', generatedAt.toISOString()],
    ['Meters', rows.value.length],
    ['Usable for the rating', `${health.value.ok} of ${health.value.total}`],
    ['Blocking the rating', health.value.blocking],
    ['Current reading', 'The meter\'s cumulative accumulator as read now, not a period consumption.'],
    ['Discarded readings', 'Recorded readings a cumulative accumulator could not have produced — ' +
      'negative, or higher than the meter reads now — excluded before any figure was derived. ' +
      'A driver or protocol fault can leave such records in history long after it is fixed.'],
    ['Resolution', 'The smallest step this meter has been observed to take, measured from runs of ' +
      'consecutive records rather than configured. Movement at or below it is treated as no ' +
      'consumption rather than as missing data, because it is the least the meter can report. ' +
      'Meters differ — some tick every 1 kWh, some every 16 — so the threshold is per meter. ' +
      'Blank means it could not be measured and the configured fallback was used.'],
    ['Status', 'Live reachability and period usability are different things. A meter can answer a ' +
      'live read and still be unusable for the rating — most often because its history does not ' +
      'reach back to the period start — in which case it blanks every dependent figure. ' +
      'Such a meter reads "No period data" with the reason in the Fault column.'],
    ['Projected hours', 'Hours of the rating period whose reading was projected forward because this ' +
      'meter failed a live read and is therefore known to be offline. This is the only substituted ' +
      'value the dashboard produces, and the only one disclosed as estimated. Blank means every hour ' +
      'rests on a recorded reading.'],
    ['Unrecorded hours', 'Hours of the rating period with no meter history behind them. Does not by ' +
      'itself mean a figure was estimated, and usually does not: history records a reading only when ' +
      'it changes, so the accumulator held its last recorded value across the stretch and the total ' +
      'is the difference between two real readings. Worth chasing all the same — a board that ought ' +
      'to be consuming and recorded nothing for days is a commissioning fault.'],
    []
  ];

  const header = [
    'Ref', 'Meter name', 'Floor', 'Zone', 'Reporting category',
    'Status', 'Current reading (kWh)', 'Resolution (kWh)', 'Discarded readings',
    'Projected hours', 'Unrecorded hours',
    'Longest gap (h)', 'Longest gap from', 'Longest gap to',
    'Fault', 'Smart Core name'
  ];

  const body = rows.value.map(row => [
    row.id.ref,
    row.id.label,
    row.id.floor,
    row.id.zone,
    props.categoryLabels[row.status.category] ?? row.status.category,
    statusLabel(row).replace(/^[^A-Za-z]+/, ''),
    row.status.ok && row.status.value !== null ? row.status.value.toFixed(1) : '',
    row.tick ?? '',
    row.rejected || '',
    // Two distinct figures: what the total rests on, and how much history is
    // simply absent. An assessor needs both, and they are often very different.
    row.hours > 0 ? Math.round(row.hours) : '',
    row.unrecorded > 0 ? Math.round(row.unrecorded) : '',
    // Dated, so an assessor querying a month can see whether a gap covers it.
    row.gap ? Math.round(row.gap.hours) : '',
    row.gap ? row.gap.from.toISOString() : '',
    row.gap ? row.gap.to.toISOString() : '',
    row.status.ok ? '' : (row.status.error ?? 'unreadable'),
    // Last, and clearly labelled as the machine name: an assessor querying a
    // figure needs the path to go and interrogate the device.
    row.name
  ]);

  downloadCsv([...preamble, header, ...body],
    `nabers-base-building-meters-${generatedAt.toISOString().slice(0, 10)}.csv`);
}
</script>

<style scoped>
.quality-wrapper {
  display:        flex;
  flex-direction: column;
  gap:            12px;
  padding:        4px 0;
}

.quality-header {
  display:         flex;
  align-items:     center;
  justify-content: space-between;
}

.quality-title {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.quality-empty {
  font-size:  14px;
  color:      rgba(255, 255, 255, 0.35);
  font-style: italic;
  padding:    8px 0;
}

code {
  background:    rgba(127, 0, 255, 0.2);
  border-radius: 4px;
  padding:       1px 5px;
  font-size:     12px;
}

.header-right {
  display:     flex;
  align-items: center;
  gap:         8px;
}

.summary {
  font-size:            12px;
  font-weight:          500;
  font-variant-numeric: tabular-nums;
}

.summary--ok {
  color: #4caf50;
}

.summary--warn {
  color: #f7a12c;
}

.quality-scroll {
  overflow-x: auto;
  /* A building that lists every board individually runs to dozens of rows, which
     is several screens. Without a cap the accordion shoves the rest of the page
     off the bottom when it opens. */
  overflow-y: auto;
  max-height: 420px;
}

.quality-table thead th {
  position: sticky;
  top:      0;
  z-index:  1;
  /* Opaque, so rows scroll under it. It cannot be the panel's own
     rgba(255,255,255,0.04), which is transparent; this is that colour already
     composited over the page background. */
  background: #16142b;
}

.quality-table {
  width:           100%;
  border-collapse: collapse;
  font-size:       13px;
}

.quality-table th {
  text-align:     left;
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.4);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding:        6px 12px 6px 0;
  border-bottom:  1px solid rgba(255, 255, 255, 0.08);
}

.quality-table td {
  padding:       8px 12px 8px 0;
  color:         rgba(255, 255, 255, 0.7);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  vertical-align: middle;
}

/* Monospace, because these are catalogue references that get read digit by digit
   and compared down the column against the contractor's schedule. */
.ref-cell {
  font-family:          monospace;
  font-size:            12px !important;
  white-space:          nowrap;
  color:                rgba(255, 255, 255, 0.5);
  font-variant-numeric: tabular-nums;
}

.meter-name {
  /* No max-width and no ellipsis. The installed titles distinguish boards late in
     the string, "Level 08 Landlords Db Lighting" against "... Db Small Power", so
     clipping at a fixed width discards the part that identifies the board. Long
     names use the horizontal scroll. */
  white-space: nowrap;
}

.name-label {
  color:       rgba(255, 255, 255, 0.85);
  font-weight: 500;
  font-size:   13px;
}

/* Floor and zone, which the titles mostly repeat but not always, and which the
   specification asks for explicitly. Secondary so it does not compete with the
   name. */
.name-location {
  display:   block;
  font-size: 11px;
  color:     rgba(255, 255, 255, 0.35);
}

.reading-cell {
  font-variant-numeric: tabular-nums;
}

.status-chip {
  display:       inline-block;
  border-radius: 4px;
  padding:       2px 8px;
  font-size:     12px;
  font-weight:   500;
}

.status-ok {
  background: rgba(76, 175, 80, 0.15);
  color:      #4caf50;
}

/* Neutral, not amber: idle is a correct, exact reading, so it must not read as a
   degraded one. Dimmer than OK only because it is worth a second look. */
.status-idle {
  background: rgba(255, 255, 255, 0.08);
  color:      rgba(255, 255, 255, 0.55);
}

.status-est {
  background: rgba(247, 161, 44, 0.15);
  color:      #f7a12c;
}

.status-err {
  background: rgba(248, 156, 155, 0.15);
  color:      #f89c9b;
}

.discard-note {
  display:     block;
  margin-top:  3px;
  font-size:   11px;
  color:       rgba(247, 161, 44, 0.75);
  white-space: nowrap;
  cursor:      help;
}

.tick-cell {
  color:       rgba(255, 255, 255, 0.45) !important;
  white-space: nowrap;
  cursor:      help;
}

.fault-cell {
  color:      rgba(248, 156, 155, 0.8) !important;
  font-size:  12px;
  min-width:  180px;
}

.summary--err {
  color: #f89c9b;
}

.gap-cell {
  color:       rgba(247, 161, 44, 0.75) !important;
  white-space: nowrap;
  cursor:      help;
}

/* One short line per point rather than a paragraph, so the panel's footnotes can
   be skimmed for the one that applies. */
.quality-desc {
  display:        flex;
  flex-direction: column;
  gap:            5px;
  font-size:      13px;
  color:          rgba(255, 255, 255, 0.35);
  line-height:    1.4;
}

.quality-desc p {
  margin: 0;
}

.quality-desc strong {
  color:       rgba(255, 255, 255, 0.6);
  font-weight: 500;
}

/* The pointer to the hover text, dimmest of the lot: it is an instruction rather
   than something to read every time. */
.desc-hint {
  color:      rgba(255, 255, 255, 0.25);
  font-style: italic;
}

.usable-desc {
  color: rgba(248, 156, 155, 0.65);
}

.usable-desc strong {
  color:       #f89c9b;
  font-weight: 500;
}

</style>
