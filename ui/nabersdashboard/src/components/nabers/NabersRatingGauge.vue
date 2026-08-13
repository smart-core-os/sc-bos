<template>
  <!--
    Two groups, always present: the rating itself, and everything supporting it.

    In `column` layout both are `display: contents`, so the children lay out
    exactly as they did before this split and the tenancy view is untouched. In
    `row` layout they become real boxes side by side, which is what lets the
    stars sit on their own row with the detail beside them rather than beneath.
  -->
  <div class="rating-gauge" :class="`rating-gauge--${layout}`">
    <div class="rating-primary">
      <!-- Inputs incomplete: name what's missing, never a fabricated figure -->
      <template v-if="missingInputs.length">
        <div class="gauge-state">Rating inputs incomplete</div>
        <div class="gauge-state-detail">
          Set {{ missingInputs.join(', ') }} in <code>{{ configSection }}</code>
          to compute a rating.
        </div>
      </template>

      <!-- No meters bound, or none readable -->
      <template v-else-if="!hasMeters">
        <div class="gauge-state">Awaiting meter data</div>
        <div class="gauge-state-detail">
          No {{ boundaryLabel.toLowerCase() }} meters are configured, so there is no
          rating to show.
        </div>
      </template>
      <!--
        Still fetching, and nothing yet to show.

        This has to outrank the states below it, because every one of them is a
        claim about data that has not arrived. The period-to-date read resolves
        two boundaries and the monthly table resolves thirteen, so the section
        renders on the first while the second is still in flight — and six days
        into a rating period that window asserted "Too early to rate" for a
        building whose twelve measured months were about to land.

        Config faults stay above it: they are knowable without any data, and
        flashing "Loading" over a misconfiguration only delays the fix.

        `rating === null` matters. Without it the daily refresh would pull a
        perfectly good rating off the screen and replace it with a spinner.
      -->
      <template v-else-if="loading && rating === null">
        <div class="gauge-state gauge-state--loading">
          <v-progress-circular indeterminate color="#7F00FF" size="28" width="3"/>
          <span>Loading</span>
        </div>
        <div class="gauge-state-detail">Reading meter history for the rating period.</div>
      </template>

      <template v-else-if="rating === null && unreadableCount > 0">
        <div class="gauge-state">Awaiting meter data</div>
        <div class="gauge-state-detail">
          {{ unreadableCount }} of {{ configuredCount }} metered end
          {{ unreadableCount === 1 ? 'use is' : 'uses are' }} unreadable — a partial
          total would understate consumption, so no rating is shown.
          <!-- Prefer the per-meter list: naming the board is actionable where
               naming the end use is not, once an end use spans many meters. -->
          <template v-if="unreadableMeterLabels.length">
            Affected: {{ unreadableMeterLabels.join('; ') }}.
          </template>
          <template v-else-if="unreadableLabels.length">
            Affected: {{ unreadableLabels.join(', ') }}.
          </template>
        </div>
      </template>
      <template v-else-if="rating === null && tooEarly">
        <div class="gauge-state">Too early to rate</div>
        <div class="gauge-state-detail">
          {{ elapsedDays }} day(s) into the rating period. Annualising this little data
          would not be meaningful, so no projection is shown yet.
        </div>
      </template>
      <template v-else-if="rating === null">
        <div class="gauge-state">Awaiting meter data</div>
        <div class="gauge-state-detail">Consumption history is still accumulating.</div>
      </template>

      <!-- The rating itself, and nothing else: the stars, the decimal figure and
           the intensity they were computed from. -->
      <template v-else>
        <div class="stars-row">
          <v-icon
              v-for="n in 6"
              :key="n"
              :icon="starIcon(n)"
              :color="starIcon(n) === 'mdi-star-outline' ? 'rgba(255,255,255,0.25)' : '#7F00FF'"
              size="46"/>
        </div>

        <div class="decimal-stars">
          {{ rating.stars.toFixed(2) }} &#9733;
          <span class="banded">({{ formatStars(rating.bandedStars) }}&#9733; banded)</span>
        </div>

        <div class="intensity-value">
          {{ rating.intensity.toFixed(1) }}
          <span class="unit"> kWhe/m² NIA/yr</span>
        </div>
      </template>
    </div>

    <div class="rating-support">
      <template v-if="showRating">
        <!-- Which of the two figures this is -->
        <div class="chip-row">
          <div class="basis-chip" :class="isProjection ? 'basis-chip--projection' : 'basis-chip--standing'">
            {{ basisChip }}
          </div>
          <!-- NABERS permits estimating missing data only where the estimation is
               disclosed, so this sits beside the headline figure rather than
               behind an accordion. -->
          <div v-if="estimatedPct !== null" class="basis-chip basis-chip--estimated">
            Includes estimated data
          </div>
        </div>

        <div v-if="basisDetail" class="basis-detail">{{ basisDetail }}</div>

        <template v-if="nextStarTarget && reductionNeeded !== null">
          <v-progress-linear
              :model-value="progressToNextStar"
              color="#7F00FF"
              bg-color="rgba(127,0,255,0.18)"
              rounded
              height="10"
              class="progress-bar"/>
          <div class="next-star-text">
            Reduce by <strong>{{ reductionNeeded.toFixed(1) }} kWhe/m²/yr</strong>
            to reach {{ formatStars(nextStarTarget.stars) }}&#9733;
          </div>
        </template>
        <!-- Only a genuine 6-star result is market leading; a missing next rung is not. -->
        <div v-else-if="rating.bandedStars >= 6" class="next-star-text next-star-text--achieved">
          Market leading performance &#10003;
        </div>

        <div v-if="benchmark !== null" class="benchmark-line">
          Benchmark {{ benchmark.toFixed(1) }} kWhe/m² &middot; BF
          {{ rating.benchmarkingFactor.toFixed(1) }}%
        </div>

        <!--
          The disclosure stays; its per-meter detail collapses. Enumerating every
          affected board ran to seven lines, which in the old 340px column
          stretched the stat cards beside it into mostly empty space. The share is
          what has to be disclosed next to the figure — which boards it came from
          is reference detail, and is listed in full with gap durations in the
          meter quality table and the CSV export.
        -->
        <div v-if="estimatedPct !== null" class="caveat">
          {{ formatPct(estimatedPct) }} of this figure's energy was projected forward past
          the last reading of an unreachable meter.
          <button
              v-if="estimatedMeterLabels.length"
              type="button"
              class="caveat-toggle"
              @click="showEstimatedMeters = !showEstimatedMeters">
            {{ showEstimatedMeters ? 'Hide' : 'Show' }} the
            {{ estimatedMeterCount }} affected {{ estimatedMeterCount === 1 ? 'meter' : 'meters' }}
          </button>
          <div v-if="showEstimatedMeters" class="caveat-detail">
            <div v-for="label in estimatedMeterLabels" :key="label">{{ label }}</div>
          </div>
        </div>

        <div v-if="pvDeductionAssumed" class="caveat">
          On-site generation is deducted in full; without export metering the
          self-consumed share is assumed, which the NABERS ruling does not permit.
        </div>
      </template>

      <div class="rating-desc">
        <strong>Indicative</strong> NABERS UK {{ boundaryLabel }} rating — the official
        method computed from this building's own meters. Not an
        accredited-Assessor-validated or lodged NABERS certificate.
      </div>
    </div>
  </div>
</template>

<script setup>
import {computed, ref} from 'vue';

const props = defineProps({
  /** @type {{stars: number, bandedStars: number, intensity: number, benchmarkingFactor: number}|null} */
  rating:              {type: Object,  default: null},
  /** Which NABERS boundary this gauge rates, for the disclaimer wording. */
  boundaryLabel:       {type: String,  default: 'Base Building'},
  /**
   * Where the missing rating inputs are configured. The two boundaries read
   * different parts of the UI config, so this cannot be hardcoded.
   */
  configSection:       {type: String,  default: 'nabersBaseBuilding'},
  /** Names of end uses whose meters could not be read. */
  unreadableLabels:    {type: Array,   default: () => []},
  /**
   * True for any annualised figure, whether from the rating period to date or
   * from the measured months behind it. The store picks whichever window is
   * usable; the gauge only distinguishes annualised from settled.
   */
  isProjection:        {type: Boolean, default: false},
  monthsOfData:        {type: Number,  default: 0},
  monthsRequired:      {type: Number,  default: 12},
  benchmark:           {type: Number,  default: null},
  missingInputs:       {type: Array,   default: () => []},
  hasMeters:           {type: Boolean, default: false},
  configuredCount:     {type: Number,  default: 0},
  /** Per-meter failures, e.g. "Terminal Fans: db-ll-n7". Preferred over labels. */
  unreadableMeterLabels: {type: Array, default: () => []},
  unreadableCount:     {type: Number,  default: 0},
  tooEarly:            {type: Boolean, default: false},
  /**
   * Whether the data behind the rating is still being fetched.
   *
   * Separate from the section's own top-level spinner, which only covers the
   * period-to-date read. The twelve-month table is a much larger fetch and lands
   * later, so there is a window where the section has rendered and the rating
   * still has nothing to work from.
   */
  loading:             {type: Boolean, default: false},
  elapsedDays:         {type: Number,  default: 0},
  nextStarTarget:      {type: Object,  default: null},
  reductionNeeded:     {type: Number,  default: null},
  progressToNextStar:  {type: Number,  default: 0},
  pvDeductionAssumed:  {type: Boolean, default: false},
  /**
   * Share of this figure's energy that came from a projected meter reading, as a
   * percentage. Null when none did — not 0, so a building with complete data
   * shows no disclosure at all rather than a redundant "0% estimated".
   */
  estimatedSharePct:   {type: Number,  default: null},
  /** Per-meter attribution, e.g. "Lifts: lift-1". */
  estimatedMeterLabels: {type: Array,  default: () => []},
  /**
   * `column` stacks everything in one narrow strip, which is what a gauge sitting
   * beside other cards in a row needs. `row` puts the stars and the intensity on
   * the left with all the supporting detail to their right, for a gauge given a
   * full-width row of its own.
   *
   * @type {'column'|'row'}
   */
  layout: {
    type:      String,
    default:   'column',
    validator: v => ['column', 'row'].includes(v)
  }
});

/**
 * Whether there is a rating to show, as opposed to one of the five states that
 * explain why there is not.
 *
 * Named once rather than repeated, because the primary and support groups have
 * to agree about it and the condition is the tail of a five-branch chain.
 */
const showRating = computed(() =>
  props.missingInputs.length === 0 && props.hasMeters && props.rating !== null
);

/**
 * The estimated share, or null when there is nothing to disclose.
 *
 * A share that rounds to 0.0% is still a share, so it is disclosed rather than
 * hidden — but a genuine zero, or an absent figure, shows nothing.
 */
const estimatedPct = computed(() =>
  (props.estimatedSharePct !== null && props.estimatedSharePct > 0)
    ? props.estimatedSharePct
    : null
);

// Two labels, not three. Whether an annualised figure came from the period to
// date or from the measured months behind it is a detail of which window was
// available; either way it is an indicative annualised estimate rather than a
// settled rating, and that is the distinction worth drawing on screen.
const basisChip = computed(() =>
  props.isProjection ? 'Projection — straight-line estimate' : '12-month standing'
);

const basisDetail = computed(() => {
  if (!props.isProjection) return '';
  return `No settled rating yet: ${props.monthsOfData} of ${props.monthsRequired} months of ` +
    'data. This figure is annualised from the data available and ignores seasonality.';
});

/** Collapsed by default: the share is the disclosure, the boards are detail. */
const showEstimatedMeters = ref(false);

// Each label is one end use with its meters, e.g. "Lifts: lift-1, lift-2", so the
// meter count is the commas across all of them rather than the label count.
const estimatedMeterCount = computed(() =>
  props.estimatedMeterLabels.reduce(
    (n, label) => n + label.slice(label.indexOf(':') + 1).split(',').length, 0)
);

/**
 * Percentages below 0.1% read as "0.0%", which looks like a bug next to a
 * disclosure that says data was estimated.
 *
 * @param {number} pct
 * @return {string}
 */
function formatPct(pct) {
  return pct < 0.1 ? '<0.1%' : `${pct.toFixed(1)}%`;
}

/**
 * Render the banded half-star figure as full / half / outline icons.
 *
 * @param {number} n the 1-based star position being drawn
 * @return {string} an mdi icon name
 */
function starIcon(n) {
  const stars = props.rating?.bandedStars ?? 0;
  if (n <= Math.floor(stars)) return 'mdi-star';
  if (n === Math.floor(stars) + 1 && stars % 1 >= 0.5) return 'mdi-star-half-full';
  return 'mdi-star-outline';
}

const formatStars = stars => (stars % 1 === 0 ? String(stars) : stars.toFixed(1));
</script>

<style scoped>
.rating-gauge {
  display:        flex;
  flex-direction: column;
  align-items:    center;
  gap:            12px;
  padding:        20px 24px;
  min-width:      320px;
  height:         100%;
  box-sizing:     border-box;
}

/* `display: contents` rather than two real boxes, so the column layout is
   exactly what it was before the groups existed — one flex column of the same
   children, with `.rating-desc`'s `margin-top: auto` still anchoring it to the
   floor of the card. */
.rating-gauge--column .rating-primary,
.rating-gauge--column .rating-support {
  display: contents;
}

/* Row layout: the rating on the left, everything supporting it on the right. */
.rating-gauge--row {
  flex-direction: row;
  align-items:    center;
  gap:            40px;
  height:         auto;
}

.rating-gauge--row .rating-primary {
  display:        flex;
  flex-direction: column;
  align-items:    center;
  gap:            8px;
  flex-shrink:    0;
}

.rating-gauge--row .rating-support {
  display:        flex;
  flex-direction: column;
  align-items:    flex-start;
  gap:            10px;
  flex:           1;
  /* Without this the long caveats refuse to wrap and force the row wider than
     the page. */
  min-width:      0;
}

/* The supporting column is a block of prose read left to right, so the centring
   that suits a narrow strip works against it here. */
.rating-gauge--row .rating-support .basis-detail,
.rating-gauge--row .rating-support .caveat,
.rating-gauge--row .rating-support .next-star-text,
.rating-gauge--row .rating-support .rating-desc,
.rating-gauge--row .rating-support .gauge-state-detail {
  text-align: left;
}

.rating-gauge--row .rating-support .rating-desc {
  margin-top: 0;
}

.rating-gauge--row .rating-support .caveat-toggle {
  margin-left: 0;
}

.stars-row {
  display: flex;
  gap:     4px;
}

.decimal-stars {
  font-size:      20px;
  font-weight:    500;
  color:          #7F00FF;
  letter-spacing: 0.04em;
  margin-top:     -4px;
  text-align:     center;
}

.banded {
  font-size:   13px;
  font-weight: 400;
  color:       rgba(255, 255, 255, 0.4);
}

.intensity-value {
  font-size:   48px;
  font-weight: 300;
  color:       #ffffff;
  line-height: 1;
  text-align:  center;
}

.unit {
  font-size: 15px;
  color:     rgba(255, 255, 255, 0.5);
}

.chip-row {
  display:         flex;
  flex-wrap:       wrap;
  gap:             6px;
  justify-content: center;
}

.basis-chip {
  font-size:      11px;
  font-weight:    500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  border-radius:  4px;
  padding:        3px 10px;
}

/* Amber, matching the estimated state used in the report and quality tables —
   distinct from the green "actual" and the red "missing". */
.basis-chip--estimated {
  background: rgba(247, 161, 44, 0.15);
  color:      #f7a12c;
}

.basis-chip--standing {
  background: rgba(76, 175, 80, 0.15);
  color:      #4caf50;
}

.basis-chip--projection {
  background: rgba(247, 161, 44, 0.15);
  color:      #f7a12c;
}

.basis-detail,
.caveat {
  font-size:   12px;
  color:       rgba(255, 255, 255, 0.4);
  text-align:  center;
  line-height: 1.4;
}

.caveat {
  color: rgba(247, 161, 44, 0.75);
}

.caveat-toggle {
  display:         block;
  margin:          4px auto 0;
  padding:         0;
  background:      none;
  border:          none;
  font:            inherit;
  color:           #f7a12c;
  text-decoration: underline;
  cursor:          pointer;
}

.caveat-detail {
  margin-top:  4px;
  font-family: monospace;
  font-size:   11px;
  text-align:  left;
  color:       rgba(247, 161, 44, 0.6);
  /* Capped rather than unbounded: a building that lists every board could put
     dozens of lines here, and the point of collapsing was to stop this column
     dictating the height of the row. */
  max-height:  120px;
  overflow-y:  auto;
}

.gauge-state {
  font-size:   28px;
  font-weight: 300;
  color:       rgba(255, 255, 255, 0.55);
  text-align:  center;
  margin-top:  24px;
}

/* The spinner sits on the same line as the word, so the transient state reads as
   in-progress rather than as one more settled condition. */
.gauge-state--loading {
  display:     flex;
  align-items: center;
  gap:         12px;
}

.gauge-state-detail {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  text-align:  center;
  line-height: 1.5;
}

code {
  background:    rgba(127, 0, 255, 0.2);
  border-radius: 4px;
  padding:       1px 5px;
  font-size:     12px;
}

.progress-bar {
  width:      100%;
  margin-top: 4px;
}

.next-star-text {
  font-size:   14px;
  color:       rgba(255, 255, 255, 0.55);
  text-align:  center;
  line-height: 1.4;
}

.next-star-text--achieved {
  color:     #4caf50;
  font-size: 15px;
}

.benchmark-line {
  font-size: 12px;
  color:     rgba(255, 255, 255, 0.3);
}

.rating-desc {
  font-size:   12px;
  color:       rgba(255, 255, 255, 0.35);
  text-align:  center;
  line-height: 1.45;
  margin-top:  auto;
}

.rating-desc strong {
  color: rgba(255, 255, 255, 0.55);
}
</style>
