<template>
  <div class="report-wrapper">
    <div class="report-header">
      <span class="report-title">Monthly NABERS data</span>
      <div class="header-right">
        <!--
          A way out to whatever page a site wants beside its monthly figures — a
          half-hourly view, a utility portal, an assessor's folder. Absent unless
          configured, and `target`/`rel` because it leaves the app: this is a
          fullscreen dashboard with no in-app way back, so replacing it in place
          would strand whoever followed the link.
        -->
        <v-btn
            v-if="linkHref"
            :href="linkHref"
            target="_blank"
            rel="noopener noreferrer"
            size="x-small"
            variant="outlined"
            color="rgba(255,255,255,0.5)"
            append-icon="mdi-open-in-new">
          {{ linkLabel }}
        </v-btn>
        <v-btn
            size="x-small"
            variant="outlined"
            color="rgba(255,255,255,0.5)"
            @click="exportCsv">
          Export CSV
        </v-btn>
      </div>
    </div>

    <div v-if="months.length === 0" class="report-empty">
      No monthly data available. Data accumulates as meter history builds.
    </div>

    <div v-else class="report-scroll">
      <table class="report-table">
        <thead>
          <tr>
            <th>Month</th>
            <th class="num-col">Gross kWh</th>
            <th class="num-col">PV kWh</th>
            <th class="num-col">Net kWh</th>
            <th class="num-col">kWh/m²</th>
            <th class="num-col">Est. kWh</th>
            <th>Data quality</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in months" :key="row.label">
            <td>{{ row.label }}</td>
            <td class="num-col">{{ row.grossKwh !== null ? Math.round(row.grossKwh).toLocaleString() : '—' }}</td>
            <td class="num-col">{{ row.pvKwh > 0 ? Math.round(row.pvKwh).toLocaleString() : '—' }}</td>
            <td class="num-col">{{ row.netKwh !== null ? Math.round(row.netKwh).toLocaleString() : '—' }}</td>
            <td class="num-col">{{ row.totalIntensity !== null ? row.totalIntensity.toFixed(2) : '—' }}</td>
            <td class="num-col">{{ estimatedCell(row) }}</td>
            <td>
              <span class="flag-chip" :class="flagClass(row)" :title="flagTitle(row)">{{ flagLabel(row) }}</span>
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="report-total">
            <td>{{ footerLabel }}</td>
            <td class="num-col">{{ Math.round(totalGross).toLocaleString() }}</td>
            <td class="num-col">{{ Math.round(totalPv).toLocaleString() }}</td>
            <td class="num-col">{{ Math.round(totalNet).toLocaleString() }}</td>
            <td class="num-col">{{ totalIntensity !== null ? totalIntensity.toFixed(2) : '—' }}</td>
            <td class="num-col">{{ totalEstimated > 0 ? Math.round(totalEstimated).toLocaleString() : '—' }}</td>
            <td/>
          </tr>
        </tfoot>
      </table>
    </div>

    <div class="report-desc">
      12-month rolling consumption data for NABERS UK annual submission. Export for review by your accredited assessor. Potential error must be disclosed for any estimated values.
      <template v-if="!isComplete">
        Totals cover the {{ monthsWithData.length }} month(s) with data and are not annualised.
      </template>
      <template v-if="totalEstimated > 0">
        <span class="estimated-note">
          {{ estimatedShareLabel }} of the reported energy ({{ Math.round(totalEstimated).toLocaleString() }} kWh)
          was projected forward past an unreachable meter's last reading and is included in the totals above.
        </span>
      </template>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {NABERS_MODEL_VERSION} from '@/util/nabersRating.js';
import {formatGap} from '@/util/meterEstimation.js';
import {downloadCsv} from '@/util/csv.js';
import {safeHttpUrl} from '@/util/externalLink.js';

const props = defineProps({
  months: {type: Array,  default: () => []},
  nia:    {type: Number, default: null},
  /**
   * `{[meterName]: {ref, label, ...}}` from device metadata, so the meters named
   * in the disclosure read as the building's own labels here and in the CSV, as
   * the EMS specification requires of every export.
   */
  meterIdentities: {type: Object, default: () => ({})},
  /**
   * Where to send a reader who wants more than the twelve rows here.
   *
   * Deliberately its own key rather than sharing the breakdown widget's `opsUrl`:
   * the useful destination beside a monthly table is rarely the same as the one
   * beside an end-use breakdown, and a single shared link would force a site to
   * pick which of the two it would rather have wrong. Empty by default, because a
   * link is only worth offering where one has been configured and a wrong
   * destination is worse than none.
   */
  linkUrl:   {type: String, default: ''},
  linkLabel: {type: String, default: 'View details'}
});

/**
 * The link target, or null when unconfigured or unusable.
 *
 * Resolved against the current document so a deployment serving the target from
 * this same origin can configure a bare path. `safeHttpUrl` rejects anything that
 * is not http(s), so a `javascript:` URL from config never reaches the anchor.
 */
const linkHref = computed(() => safeHttpUrl(props.linkUrl, window.location.href));

/**
 * A meter's human-readable name, or the tail of its Smart Core name if metadata
 * has not arrived.
 *
 * @param {string} name
 * @return {string}
 */
function meterName(name) {
  const id = props.meterIdentities[name];
  if (!id) return name.slice(name.lastIndexOf('/') + 1);
  return id.ref ? `${id.ref} ${id.label}` : id.label;
}

// Every footer figure is a sum over the SAME set of months — the ones that have
// data. Mixing a partial-month sum with an annualised intensity would misreport
// both, so when the year is incomplete the footer says so and reports the
// measured total rather than silently annualising one column and not the others.
const monthsWithData = computed(() => props.months.filter(r => r.hasData && r.netKwh !== null));

const isComplete = computed(() =>
  props.months.length === 12 && monthsWithData.value.length === 12
);

const totalGross = computed(() => monthsWithData.value.reduce((a, r) => a + r.grossKwh, 0));
const totalPv    = computed(() => monthsWithData.value.reduce((a, r) => a + (r.pvKwh ?? 0), 0));
const totalNet   = computed(() => monthsWithData.value.reduce((a, r) => a + r.netKwh, 0));

/** Energy in the totals above that came from a projected reading. */
const totalEstimated = computed(() =>
  monthsWithData.value.reduce((a, r) => a + (r.estimatedKwh ?? 0), 0)
);

const estimatedSharePct = computed(() =>
  totalNet.value > 0 ? (totalEstimated.value / totalNet.value) * 100 : 0
);

const estimatedShareLabel = computed(() =>
  estimatedSharePct.value < 0.1 ? 'Less than 0.1%' : `${estimatedSharePct.value.toFixed(1)}%`
);

/** Measured intensity over the months present — annualised only when all 12 are. */
const totalIntensity = computed(() => {
  if (!monthsWithData.value.length || !props.nia) return null;
  return totalNet.value / props.nia;
});

const footerLabel = computed(() =>
  isComplete.value
    ? '12-month total'
    : `${monthsWithData.value.length}-month total (partial)`
);

/**
 * @param {Object} row
 * @return {string}
 */
function flagClass(row) {
  if (!row.hasData) return 'flag-missing';
  return row.quality === 'estimated' ? 'flag-estimated' : 'flag-actual';
}

/**
 * @param {Object} row
 * @return {string}
 */
function flagLabel(row) {
  if (!row.hasData) return '✗ Missing';
  return row.quality === 'estimated' ? '~ Estimated' : '✓ Actual';
}

/**
 * Which meters were projected or withheld and why, on hover.
 *
 * Names the board rather than just the month: over seventeen distribution
 * boards "March was estimated" is not something an engineer can act on. The
 * missing case matters at least as much — a bare "✗ Missing" beside a month the
 * month-end report has figures for reads as this dashboard being broken, when what
 * actually happened is that one named board did something a cumulative meter
 * cannot do.
 *
 * @param {Object} row
 * @return {string|undefined}
 */
function flagTitle(row) {
  if (!row.hasData) {
    // One line per board, each with its own reason, because a month can be
    // withheld by several meters doing different things and a single shared
    // sentence hid which was which.
    const lines = (row.failures ?? []).map(f => `${meterName(f.name)} — ${f.reason}`);
    if (lines.length) return lines.join('\n');
    const names = (row.unreadableMeters ?? []).map(meterName);
    const who = names.length ? names.join(', ') : 'one or more meters';
    return row.reason ? `${who}: ${row.reason}` : `No figure for ${who}`;
  }
  if (row.quality !== 'estimated') return undefined;
  const names = (row.estimatedMeters ?? []).map(meterName);
  const gap = formatGap(row.estimatedHours ?? 0);
  const who = names.length ? `Projected: ${names.join(', ')}` : 'Projected meter data';
  return gap ? `${who} — longest gap ${gap}` : who;
}

/**
 * @param {Object} row
 * @return {string}
 */
function estimatedCell(row) {
  if (!row.hasData || !(row.estimatedKwh > 0)) return '—';
  return Math.round(row.estimatedKwh).toLocaleString();
}


/**
 * Export the table for an accredited assessor.
 *
 * This file is the reporting contract, so it carries the things the screen
 * carries and the screen's provenance too: which constant set produced the
 * figures, when they were taken, and an explicit statement of what was
 * estimated. NABERS permits estimated data only where it is disclosed, and a
 * spreadsheet detached from this dashboard is exactly where that disclosure
 * would otherwise be lost.
 */
function exportCsv() {
  const generatedAt = new Date();
  const estimatedRows = props.months.filter(r => r.hasData && (r.estimatedKwh ?? 0) > 0);

  const preamble = [
    ['NABERS UK base building — monthly consumption'],
    ['Generated', generatedAt.toISOString()],
    ['Model version', NABERS_MODEL_VERSION],
    ['Rated area (m² NIA)', props.nia ?? ''],
    ['Indicative', 'Computed from this building\'s own meters. Not an accredited-Assessor-validated or lodged NABERS certificate.'],
    ['Estimated data', estimatedRows.length
      ? `${estimatedShareLabel.value} of the reported energy (${Math.round(totalEstimated.value)} kWh) ` +
        'was projected forward past the last reading of a meter that failed a live read and is ' +
        'therefore known to be offline. The projection runs at that meter\'s own mean rate, inflated ' +
        'by a configured uplift so a substituted value cannot understate consumption, and is refused ' +
        'outright once the silence exceeds the estimation window. Months affected are marked ' +
        '"Estimated" below.'
      : 'None. Every figure below is derived from actual meter readings.'],
    ['Missing readings', 'Where a meter recorded nothing either side of a month boundary, its ' +
      'accumulator value at that boundary is the last reading before it, carried forward. This is a ' +
      'measurement rather than an estimate: the history automation records a reading only when it ' +
      'changes, so a stretch with no records is a stretch in which every poll read that same value. ' +
      'Consumption is therefore attributed to the month in which the meter was next seen to move, ' +
      'and the total across any set of whole months is unaffected by where the boundary fell.'],
    []
  ];

  const header = [
    'Month', 'Gross kWh', 'PV kWh', 'Net kWh', 'kWh/m²',
    'Estimated kWh', 'Estimated %', 'Longest gap (h)', 'Estimated meters', 'Data quality',
    // An assessor reading a blank row needs the same answer the tooltip gives, and
    // a spreadsheet detached from this dashboard is exactly where "why is December
    // empty" otherwise becomes unanswerable.
    'Affected meters', 'Reason'
  ];

  const rows = props.months.map(r => [
    r.label,
    r.grossKwh !== null ? Math.round(r.grossKwh) : '',
    r.pvKwh > 0 ? Math.round(r.pvKwh) : '',
    r.netKwh !== null ? Math.round(r.netKwh) : '',
    r.totalIntensity !== null ? r.totalIntensity.toFixed(2) : '',
    (r.estimatedKwh ?? 0) > 0 ? Math.round(r.estimatedKwh) : '',
    (r.estimatedKwh ?? 0) > 0 ? (r.estimatedPct ?? 0).toFixed(1) : '',
    (r.estimatedHours ?? 0) > 0 ? Math.round(r.estimatedHours) : '',
    (r.estimatedMeters ?? []).map(meterName).join('; '),
    flagLabel(r).replace(/^[^A-Za-z]+/, ''),
    (r.unreadableMeters ?? []).map(meterName).join('; '),
    // Per board, so a month withheld by several meters is not reduced to whichever
    // one happened to be first in config order.
    (r.failures ?? []).length
      ? r.failures.map(f => `${meterName(f.name)}: ${f.reason}`).join(' | ')
      : (r.reason ?? '')
  ]);

  // The screen has always shown a total row and the CSV never did, so the two
  // artefacts disagreed on scope. It is stated as covering only the months with
  // data, exactly as the footer on screen does.
  const total = [
    footerLabel.value,
    Math.round(totalGross.value),
    Math.round(totalPv.value),
    Math.round(totalNet.value),
    totalIntensity.value !== null ? totalIntensity.value.toFixed(2) : '',
    totalEstimated.value > 0 ? Math.round(totalEstimated.value) : '',
    totalEstimated.value > 0 ? estimatedSharePct.value.toFixed(1) : '',
    '', '', '', '', ''
  ];

  downloadCsv([...preamble, header, ...rows, total],
    `nabers-base-building-${generatedAt.toISOString().slice(0, 7)}.csv`);
}
</script>

<style scoped>
.report-wrapper {
  display:        flex;
  flex-direction: column;
  gap:            12px;
  padding:        4px 0;
}

.report-header {
  display:         flex;
  align-items:     center;
  justify-content: space-between;
  gap:             12px;
}

.header-right {
  display:     flex;
  align-items: center;
  gap:         12px;
}

.report-title {
  font-size:      12px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.report-empty {
  font-size:  14px;
  color:      rgba(255, 255, 255, 0.35);
  font-style: italic;
  padding:    8px 0;
}

.report-scroll {
  overflow-x: auto;
}

.report-table {
  width:           100%;
  border-collapse: collapse;
  font-size:       13px;
}

.report-table th {
  text-align:     left;
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.4);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding:        6px 16px 6px 0;
  border-bottom:  1px solid rgba(255, 255, 255, 0.08);
  white-space:    nowrap;
}

.report-table td {
  padding:       8px 16px 8px 0;
  color:         rgba(255, 255, 255, 0.7);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.num-col {
  text-align:           right !important;
  font-variant-numeric: tabular-nums;
  padding-right:        24px !important;
}

.report-total td {
  font-weight:   500;
  color:         #ffffff !important;
  border-top:    1px solid rgba(255, 255, 255, 0.15);
  border-bottom: none !important;
  padding-top:   12px !important;
}

.flag-chip {
  display:       inline-block;
  border-radius: 4px;
  padding:       2px 8px;
  font-size:     12px;
  font-weight:   500;
}

.flag-actual {
  background: rgba(76, 175, 80, 0.15);
  color:      #4caf50;
}

.flag-estimated {
  background: rgba(247, 161, 44, 0.15);
  color:      #f7a12c;
  cursor:     help;
}

.flag-missing {
  background: rgba(248, 156, 155, 0.15);
  color:      #f89c9b;
  cursor:     help;
}

.report-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  line-height: 1.4;
}

.estimated-note {
  color: rgba(247, 161, 44, 0.75);
}
</style>
