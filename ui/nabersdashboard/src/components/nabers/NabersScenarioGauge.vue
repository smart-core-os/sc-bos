<template>
  <div class="gauge-wrapper">
    <div class="gauge-header">
      <div class="gauge-title">Scenario proximity</div>
      <button v-if="selectedId" type="button" class="clear-btn" @click="selectedId = null">
        Clear
      </button>
    </div>

    <div class="gauge-track-area">
      <div class="gauge-track" :class="{'gauge-track--limited': axisIsLimit}">
        <!-- filled bar coloured by zone -->
        <div
            v-if="currentIntensity !== null"
            class="gauge-fill"
            :style="{ width: currentPct + '%', background: fillColor }"/>

        <!--
          One tick per marker, and a label only on the active one. Labelling every
          marker put "S02", "S04" and "S05" — 58.7, 59.3 and 59.8 on a 90-wide
          axis — inside two pixels of each other, so the ids overlapped into an
          unreadable smear. Picking one out of the list below labels it here.
        -->
        <div
            v-for="m in markers"
            :key="m.id"
            class="ref-line"
            :class="{
              'ref-line--dfp': m.kind === 'dfp',
              'ref-line--active': m.id === activeId,
              'ref-line--muted': activeId !== null && m.id !== activeId
            }"
            :style="{ left: m.pct + '%' }">
          <span v-if="m.id === activeId" class="ref-label" :class="{'ref-label--dfp': m.kind === 'dfp'}">
            {{ m.id }} · {{ m.value }}
          </span>
        </div>

        <!-- Current value pointer -->
        <div
            v-if="currentIntensity !== null"
            class="current-pointer"
            :style="{ left: currentPct + '%' }"/>
      </div>

      <!-- When the axis is capped at the target rating's ceiling the right-hand
           figure is a real threshold, so it is named. Left unnamed it invites
           exactly the wrong reading: the padded maximum used to print 70 where
           the 5-star limit is 76.0. -->
      <div class="gauge-axis">
        <span>0</span>
        <span :class="{'axis-limit': axisIsLimit}">
          {{ axisLabel }}
        </span>
      </div>
    </div>

    <!-- What picking a marker actually tells you: where the building sits against
         it. Reserved height, so selecting does not shift the list below. -->
    <div class="selection-note" :class="{'selection-note--empty': !activeMarker}">
      <template v-if="activeMarker && activeDelta !== null">
        <strong>{{ activeMarker.id }}</strong> {{ activeMarker.label }} —
        current {{ currentIntensity.toFixed(1) }} is
        <span :class="activeDelta <= 0 ? 'delta-under' : 'delta-over'">
          {{ Math.abs(activeDelta).toFixed(1) }} kWh/m² {{ activeDelta <= 0 ? 'below' : 'above' }}
        </span>
        it.
      </template>
      <template v-else-if="activeMarker">
        <strong>{{ activeMarker.id }}</strong> {{ activeMarker.label }} —
        {{ activeMarker.value }} kWh/m². No current figure to compare against.
      </template>
      <template v-else>Select a scenario to place it on the track.</template>
    </div>

    <!-- Legend, and the selector -->
    <div class="scenario-list">
      <button
          v-for="m in markers"
          :key="m.id"
          type="button"
          class="scenario-item"
          :class="{'scenario-item--selected': m.id === selectedId, 'scenario-item--dfp': m.kind === 'dfp'}"
          :aria-pressed="m.id === selectedId"
          @click="toggle(m.id)"
          @mouseenter="hoveredId = m.id"
          @mouseleave="hoveredId = null"
          @focus="hoveredId = m.id"
          @blur="hoveredId = null">
        <span class="scenario-id">{{ m.id }}</span>
        <span class="scenario-desc">{{ m.label }}</span>
        <span class="scenario-val">{{ m.value }} kWh/m²</span>
      </button>
    </div>

    <div class="gauge-card-desc">
      Current energy intensity relative to the off-axis scenarios modelled in the building's
      Design for Performance assessment. {{ legendText }}
      <template v-if="beyondEnvelope">
        Currently past every modelled scenario, which the colour deliberately does not
        treat as failure: the whole modelled spread is worth a fraction of a star.
      </template>
    </div>
  </div>
</template>

<script setup>
import {computed, ref} from 'vue';
import {
  DFP_RECOMMENDED_MARGIN_PCT, DFP_SEVERITY_COLOR, dfpSeverity, dfpSeverityColor
} from '@/util/dfpSeverity.js';
import {headroomPct} from '@/util/nabersRating.js';

const props = defineProps({
  currentIntensity: {type: Number, default: null},
  dfpTotal:         {type: Number, default: null},
  scenarios:        {type: Array,  default: () => []},
  /**
   * What the reference line is, for the legend row. The base building boundary
   * compares against a DfP modelled total; the tenancy boundary against its
   * benchmark, so the two want different words for the same marker.
   */
  referenceId:      {type: String, default: 'DfP'},
  referenceLabel:   {type: String, default: 'Design for Performance target'},
  /**
   * The intensity ceiling of the rating the building is targeting, kWhe/m²·pa.
   *
   * Supplying it changes the gauge in two ways: the axis is capped there, so the
   * unfilled part of the track *is* the headroom the stat card reports; and the
   * colour grades on that headroom rather than on the modelled envelope. Absent
   * — the tenancy boundary has no target rating — the gauge keeps its original
   * padded axis and envelope colouring.
   */
  targetStarMax:    {type: Number, default: null},
  /** What to call that ceiling on the axis, e.g. "5★ limit". */
  targetStarLabel:  {type: String, default: ''},
  recommendedMarginPct: {type: Number, default: DFP_RECOMMENDED_MARGIN_PCT}
});

/** The picked marker, or null. Nothing is picked until someone picks it. */
const selectedId = ref(null);

/** Hovering a legend row previews it on the track without committing. */
const hoveredId = ref(null);

/**
 * @param {string} id
 */
function toggle(id) {
  selectedId.value = selectedId.value === id ? null : id;
}

/**
 * Drop a leading repeat of the id from a label.
 *
 * The id is always rendered beside the label, so a reference named "DfP" whose
 * label is "DfP Stage 4 modelled total" would read "DfP DfP Stage 4 modelled
 * total". Callers should not have to word around that.
 *
 * @param {string} id
 * @param {string} label
 * @return {string}
 */
function withoutIdPrefix(id, label) {
  if (!id || !label || !label.startsWith(id)) return label;
  const rest = label.slice(id.length).trimStart();
  return rest.length ? rest : label;
}

const sortedScenarios = computed(() =>
  [...props.scenarios].sort((a, b) => a.kwhPerM2 - b.kwhPerM2)
);

const peakValue = computed(() => {
  const vals = props.scenarios.map(s => s.kwhPerM2);
  if (props.dfpTotal !== null) vals.push(props.dfpTotal);
  if (props.currentIntensity !== null) vals.push(props.currentIntensity);
  return Math.max(...vals, 0);
});

/**
 * Whether the axis ends on the target rating's ceiling rather than on padding.
 *
 * Only when everything plotted fits below it. Past the ceiling the cap would pin
 * the pointer to the right edge and stop it moving, which is the one case where
 * you most want to see how far past it has gone.
 */
const axisIsLimit = computed(() =>
  Number.isFinite(props.targetStarMax) &&
  props.targetStarMax > 0 &&
  props.targetStarMax >= peakValue.value
);

const maxValue = computed(() => {
  if (axisIsLimit.value) return props.targetStarMax;
  // Padded fallback. Never return 0 — the axis divides by this.
  return peakValue.value > 0 ? Math.ceil((peakValue.value * 1.15) / 10) * 10 : 10;
});

const formatAxis = v => (Number.isInteger(v) ? String(v) : v.toFixed(1));

const axisLabel = computed(() => {
  const figure = `${formatAxis(maxValue.value)} kWh/m²/yr`;
  if (!axisIsLimit.value || !props.targetStarLabel) return figure;
  return `${figure} · ${props.targetStarLabel}`;
});

const toPercent = val => Math.min(100, Math.max(0, (val / maxValue.value) * 100));

const currentPct = computed(() => props.currentIntensity !== null ? toPercent(props.currentIntensity) : 0);

/**
 * The reference target and every scenario as one ordered set.
 *
 * One list, because the reference line collided with the nearest scenario label
 * just as readily as the scenarios collided with each other — S01 at 53.8 sits
 * within two pixels of a 55.7 target. Treating it as another selectable marker
 * means nothing on the track is labelled unless it was asked for.
 */
const markers = computed(() => {
  const items = sortedScenarios.value.map(s => ({
    id: s.id, label: s.label, value: s.kwhPerM2, kind: 'scenario'
  }));
  if (props.dfpTotal !== null) {
    items.push({
      id: props.referenceId, label: props.referenceLabel, value: props.dfpTotal, kind: 'dfp'
    });
  }
  return items
    .sort((a, b) => a.value - b.value)
    .map(m => ({...m, pct: toPercent(m.value), label: withoutIdPrefix(m.id, m.label)}));
});

/** Hover wins over selection, so previewing a row does not lose the pick. */
const activeId = computed(() => hoveredId.value ?? selectedId.value);

const activeMarker = computed(() => markers.value.find(m => m.id === activeId.value) ?? null);

const activeDelta = computed(() =>
  (activeMarker.value && props.currentIntensity !== null)
    ? props.currentIntensity - activeMarker.value.value
    : null
);

const worstScenario = computed(() =>
  sortedScenarios.value.length ? sortedScenarios.value[sortedScenarios.value.length - 1].kwhPerM2 : null
);

const beyondEnvelope = computed(() =>
  props.currentIntensity !== null &&
  worstScenario.value !== null &&
  props.currentIntensity > worstScenario.value
);

/** Headroom below the target rating's ceiling, when there is one to measure. */
const currentHeadroomPct = computed(() =>
  headroomPct(props.currentIntensity, props.targetStarMax)
);

/**
 * With a target rating, the shared ladder in util/dfpSeverity.js — so this gauge
 * and the stat cards above it cannot disagree about the same building.
 *
 * Without one, the original envelope ladder: at or under the design target,
 * inside the modelled off-axis envelope, or beyond everything modelled. Note the
 * ordering, which is load-bearing — an earlier version tested the DfP target
 * first and, since every scenario sits above that target, made the middle band
 * unreachable.
 */
const fillColor = computed(() => {
  if (props.currentIntensity === null) return '#7F00FF';
  if (currentHeadroomPct.value === null) {
    if (props.dfpTotal !== null && props.currentIntensity <= props.dfpTotal) return DFP_SEVERITY_COLOR.good;
    if (!beyondEnvelope.value && worstScenario.value !== null) return DFP_SEVERITY_COLOR.watch;
    return DFP_SEVERITY_COLOR.risk;
  }
  return dfpSeverityColor(dfpSeverity({
    intensity:            props.currentIntensity,
    designTarget:         props.dfpTotal,
    headroomPct:          currentHeadroomPct.value,
    recommendedMarginPct: props.recommendedMarginPct
  }));
});

const legendText = computed(() => {
  if (currentHeadroomPct.value === null) {
    return 'Green = at or below the design target; amber = above target but within the ' +
      'modelled off-axis envelope; red = beyond every modelled scenario.';
  }
  const limit = props.targetStarLabel || 'the target rating';
  return 'Green = at or below the design target; amber = above it but still holding the ' +
    `${props.recommendedMarginPct}% headroom below ${limit} that DfP recommends; ` +
    'red = that headroom lost, so the rating itself is at risk.';
});
</script>

<style scoped>
.gauge-wrapper {
  background:     rgba(255, 255, 255, 0.05);
  border:         1px solid rgba(255, 255, 255, 0.08);
  border-radius:  12px;
  padding:        20px 24px;
  display:        flex;
  flex-direction: column;
  gap:            16px;
  flex:           2;
}

.gauge-header {
  display:         flex;
  align-items:     baseline;
  justify-content: space-between;
  gap:             12px;
}

.gauge-title {
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.clear-btn {
  background:      none;
  border:          none;
  padding:         0;
  font:            inherit;
  font-size:       11px;
  color:           rgba(255, 255, 255, 0.5);
  text-transform:  uppercase;
  letter-spacing:  0.08em;
  text-decoration: underline;
  cursor:          pointer;
}

.clear-btn:hover {
  color: rgba(255, 255, 255, 0.8);
}

.gauge-track-area {
  display:        flex;
  flex-direction: column;
  gap:            4px;
}

.gauge-track {
  position:      relative;
  height:        40px;
  background:    rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  overflow:      visible;
  /* Headroom for the active marker's label, which sits above the bar. Reserved
     unconditionally so selecting one does not nudge the track downwards. */
  margin-top:    20px;
}

/* The track's right edge is the target rating's ceiling, so it is drawn as a
   threshold rather than as the end of a box. */
.gauge-track--limited {
  border-right: 2px solid rgba(248, 156, 155, 0.7);
}

.gauge-fill {
  position:      absolute;
  top:           0;
  left:          0;
  height:        100%;
  border-radius: 4px;
  transition:    width 0.6s ease, background 0.4s ease;
}

.ref-line {
  position:   absolute;
  top:        -6px;
  bottom:     -6px;
  width:      2px;
  background: rgba(255, 255, 255, 0.35);
  transform:  translateX(-50%);
  transition: background 0.2s ease, top 0.2s ease, bottom 0.2s ease;
}

.ref-line--dfp {
  background: rgba(247, 161, 44, 0.8);
  width:      2px;
}

/* The active marker grows past the track and brightens, so it reads as the one
   being talked about even where three ticks sit on top of each other. */
.ref-line--active {
  background: #ffffff;
  top:        -14px;
  bottom:     -10px;
  z-index:    2;
}

.ref-line--dfp.ref-line--active {
  background: #f7a12c;
}

/* Everything not active recedes rather than disappearing: the distribution of
   scenarios is worth seeing even when one is picked out of it. */
.ref-line--muted {
  background: rgba(255, 255, 255, 0.18);
}

.ref-line--dfp.ref-line--muted {
  background: rgba(247, 161, 44, 0.35);
}

.ref-label {
  position:    absolute;
  top:         -30px;
  left:        50%;
  transform:   translateX(-50%);
  font-size:   11px;
  font-weight: 500;
  color:       #ffffff;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.ref-label--dfp {
  color: #f7a12c;
}

.current-pointer {
  position:   absolute;
  top:        -4px;
  bottom:     -4px;
  width:      3px;
  background: #ffffff;
  border-radius: 2px;
  transform:  translateX(-50%);
  box-shadow: 0 0 6px rgba(255, 255, 255, 0.6);
}

.gauge-axis {
  display:         flex;
  justify-content: space-between;
  font-size:       10px;
  color:           rgba(255, 255, 255, 0.3);
  padding:         0 2px;
}

/* A named threshold, not a scale maximum, so it is legible rather than faint. */
.axis-limit {
  color:       rgba(248, 156, 155, 0.85);
  font-weight: 500;
}

.selection-note {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.6);
  line-height: 1.4;
  /* Two lines' worth, reserved: the note appears and disappears with the
     selection, and without a floor the whole list below it would jump. */
  min-height:  36px;
}

.selection-note--empty {
  color:      rgba(255, 255, 255, 0.25);
  font-style: italic;
}

.selection-note strong {
  color:       rgba(255, 255, 255, 0.85);
  font-weight: 600;
  margin-right: 2px;
}

.delta-under {
  color: #4caf50;
}

.delta-over {
  color: #f89c9b;
}

.scenario-list {
  display:        flex;
  flex-direction: column;
  gap:            2px;
}

.scenario-item {
  display:       flex;
  align-items:   baseline;
  gap:           8px;
  width:         100%;
  text-align:    left;
  background:    none;
  border:        none;
  border-radius: 4px;
  padding:       4px 6px;
  margin:        0 -6px;
  font:          inherit;
  cursor:        pointer;
  transition:    background 0.15s ease;
}

.scenario-item:hover {
  background: rgba(255, 255, 255, 0.06);
}

.scenario-item:focus-visible {
  outline:        2px solid rgba(127, 0, 255, 0.8);
  outline-offset: 1px;
}

.scenario-item--selected {
  background: rgba(255, 255, 255, 0.1);
}

.scenario-item--selected .scenario-id,
.scenario-item--selected .scenario-desc,
.scenario-item--selected .scenario-val {
  color: rgba(255, 255, 255, 0.9);
}

.scenario-id {
  font-size:   13px;
  font-weight: 600;
  color:       rgba(255, 255, 255, 0.6);
  min-width:   40px;
}

.scenario-item--dfp .scenario-id {
  color: rgba(247, 161, 44, 0.9);
}

.scenario-desc {
  font-size: 13px;
  color:     rgba(255, 255, 255, 0.4);
  flex:      1;
}

.scenario-val {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.3);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.gauge-card-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  auto;
  line-height: 1.4;
}
</style>
