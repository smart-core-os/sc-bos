<template>
  <div v-if="enabled" class="nabers-base-section">
    <!-- Header -->
    <div class="section-header">
      <h2 class="section-title">NABERS Base Building Energy</h2>
      <router-link to="/" class="nav-link">
        <v-btn variant="text" size="small" color="rgba(255,255,255,0.45)">
          ← Tenant view
        </v-btn>
      </router-link>
    </div>

    <div v-if="store.loading" class="state-wrapper">
      <v-progress-circular indeterminate color="#7F00FF" size="48"/>
    </div>

    <div v-else-if="store.error" class="state-wrapper">
      <v-alert type="error" :text="String(store.error)" variant="tonal"/>
    </div>

    <div v-else class="nabers-body">
      <!--
        The disclosure NABERS requires, above the fold rather than behind an
        accordion: anyone reading a figure off this screen has to be able to see
        that some of it was estimated without going looking for the caveat.
      -->
      <div v-if="store.hasEstimatedData" class="estimation-banner">
        <v-icon icon="mdi-chart-timeline-variant" size="16"/>
        <span>
          <strong>Includes estimated data.</strong>
          {{ estimatedShareLabel }} of the energy behind the figures below was estimated rather
          than measured. {{ estimationMechanism(store.estimatedKind) }}
          Estimated months are marked in the monthly report and dashed on the trend chart.
        </span>
      </div>

      <!-- Row 1: the rating on its own row, with its supporting detail beside it
           rather than stacked beneath it in a 340px column. -->
      <div class="row row--rating">
        <NabersRatingGauge
            layout="row"
            boundary-label="Base Building"
            :rating="store.headlineRating"
            :is-projection="store.headlineIsProjection"
            :months-of-data="store.monthsOfData"
            :benchmark="store.benchmark"
            :missing-inputs="store.missingInputs"
            :has-meters="store.hasConfiguredMeters"
            :configured-count="store.configuredCategories.length"
            :unreadable-count="store.unreadableCategories.length"
            :unreadable-labels="store.unreadableLabels"
            :unreadable-meter-labels="store.unreadableMeterLabels"
            :too-early="store.hasConfiguredMeters && !store.canProject && !store.canUseTrailing"
            :loading="store.monthlyLoading"
            :elapsed-days="store.elapsedDays"
            :next-star-target="store.nextStarTarget"
            :reduction-needed="store.reductionNeeded"
            :progress-to-next-star="store.progressToNextStar"
            :pv-deduction-assumed="store.pvDeductionAssumed"
            :estimated-share-pct="store.estimatedShare"
            :estimated-kind="store.estimatedKind"
            :estimated-meter-labels="store.estimatedMeterLabels"/>
      </div>

      <!-- Row 2: stat cards -->
      <div class="row row--stats">
        <NabersStatCard
            title="vs DfP target"
            :value="dfpDiffLabel"
            :value-color="dfpDiffColor"
            :empty-reason="dfpDiffReason"
            :subtitle="`Annualised net base building intensity vs the ${dfpTargetSource} modelled total`"/>
        <NabersStatCard
            title="Carbon intensity"
            :value="carbonValue"
            unit="kgCO₂e/m²/yr"
            :empty-reason="carbonReason"
            subtitle="CO₂e from electricity use, per floor area. Reported separately — emission factors do not enter the NABERS rating"/>
        <NabersStatCard
            :title="headroomTitle"
            :value="headroomLabel"
            :value-color="headroomColor"
            :empty-reason="headroomReason"
            :subtitle="headroomSubtitle"/>
        <NabersStatCard
            :title="stretchTitle"
            :value="stretchLabel"
            :value-color="stretchColor"
            :empty-reason="stretchReason"
            :subtitle="stretchSubtitle"/>
      </div>

      <!-- Row 3: category breakdown chart + monthly trend -->
      <div class="row row--charts">
        <div class="chart-breakdown">
          <NabersBaseBreakdownChart
              :category-intensities="store.categoryIntensities"
              :dfp-targets="store.dfpTargets"
              :categories="store.categories"
              :category-labels="store.categoryLabels"
              :category-estimated-kwh="store.categoryEstimatedKwh"
              :category-measured-kwh="store.categoryBasisKwh"
              :intensity-basis="store.intensityBasis ?? 'period'"
              :months-of-data="store.monthsOfData"
              :trailing-days-covered="store.trailingDaysCovered"
              :nia="store.nia"
              :elapsed-days="store.elapsedDays"
              :target-label="dfpTargetLabel"
              :ops-url="bbCfg.opsUrl ?? ''"
              :ops-link-label="bbCfg.opsLinkLabel ?? 'Detailed energy view'"/>
        </div>
        <div class="chart-trend">
          <NabersBaseTrendChart
              :months="store.monthlyData"
              :five-star-max="store.fiveStarMax"
              :four-star-max="store.fourStarMax"/>
        </div>
      </div>

      <!-- Row 4: off-axis scenario gauge -->
      <div class="row row--gauge">
        <NabersScenarioGauge
            :current-intensity="store.totalIntensity"
            :dfp-total="store.dfpTargets.total ?? null"
            :reference-label="dfpTargetSource + ' modelled total'"
            :scenarios="store.scenarios"
            :target-star-max="store.targetStarMax"
            :target-star-label="targetStarLabel"
            :recommended-margin-pct="store.recommendedMarginPct"/>
      </div>

      <!-- Row 5: occupancy + energy source + cert timeline -->
      <div class="row row--bottom">
        <NabersBaseOccupancyTracker
            :building-type="bbCfg.buildingType ?? 'existing'"
            :nla-let-pct="bbCfg.nlaLetPct ?? null"
            :rating-period-start="ratingPeriodStartIso"
            :occupancy-certificate-date="bbCfg.occupancyCertificateDate ?? ''"/>
        <NabersBaseEnergySourceChart
            :gross-kwh="grossKwhPerM2"
            :pv-kwh="store.pvIntensity"
            :intensity-basis="store.intensityBasis ?? 'period'"
            :months-of-data="store.monthsOfData"
            :trailing-days-covered="store.trailingDaysCovered"
            :elapsed-days="store.elapsedDays"
            :dfp-pv-intensity="store.dfpPvIntensity"
            :dfp-pv-share-pct="store.dfpPvSharePct"/>
        <NabersBaseCertTimeline
            :cert-issue-date="bbCfg.certificateIssueDate ?? ''"
            :occupancy-certificate-date="bbCfg.occupancyCertificateDate ?? ''"
            :assessment-type="bbCfg.assessmentType ?? ''"/>
      </div>

      <!-- Row 6: accordions -->
      <v-expansion-panels variant="accordion" class="accordions">
        <v-expansion-panel bg-color="rgba(255,255,255,0.04)">
          <v-expansion-panel-title class="panel-title">
            Monthly NABERS report
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <NabersBaseMonthlyReport
                :months="store.monthlyData"
                :nia="store.nia"
                :meter-identities="store.meterIdentities"
                :link-url="bbCfg.monthlyReportUrl ?? ''"
                :link-label="bbCfg.monthlyReportLinkLabel ?? 'View details'"/>
          </v-expansion-panel-text>
        </v-expansion-panel>

        <v-expansion-panel bg-color="rgba(255,255,255,0.04)">
          <v-expansion-panel-title class="panel-title">
            Meter data quality
          </v-expansion-panel-title>
          <v-expansion-panel-text>
            <NabersBaseMeterQuality
                :meter-statuses="store.meterStatuses"
                :category-labels="store.categoryLabels"
                :meter-identities="store.meterIdentities"
                :meter-estimation="store.meterEstimation"
                :meter-idle="store.meterIdle"
                :meter-failure-reasons="store.meterFailureReasons"
                @refresh="store.refreshMeterStatuses()"/>
          </v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>
    </div>
  </div>

  <div v-else class="nabers-base-section state-wrapper">
    <v-alert
        type="info"
        text="Base building dashboard is disabled. Set nabersBaseBuilding.enabled = true in dashboards-config.json."
        variant="tonal"/>
  </div>
</template>

<script setup>
import {computed, onMounted, onUnmounted} from 'vue';
import {format} from 'date-fns';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {useNabersBaseBuildingStore} from '@/stores/nabersBaseBuildingMetrics.js';
import {
  DFP_SEVERITY_COLOR, dfpSeverity, dfpSeverityColor, headroomSeverity
} from '@/util/dfpSeverity.js';
import {estimationMechanism} from '@/util/disclosure.js';
import NabersStatCard            from '@/components/nabers/NabersStatCard.vue';
import NabersScenarioGauge       from '@/components/nabers/NabersScenarioGauge.vue';
import NabersRatingGauge         from '@/components/nabers/NabersRatingGauge.vue';
import NabersBaseBreakdownChart  from './NabersBaseBreakdownChart.vue';
import NabersBaseTrendChart      from './NabersBaseTrendChart.vue';
import NabersBaseEnergySourceChart from './NabersBaseEnergySourceChart.vue';
import NabersBaseOccupancyTracker  from './NabersBaseOccupancyTracker.vue';
import NabersBaseCertTimeline    from './NabersBaseCertTimeline.vue';
import NabersBaseMonthlyReport   from './NabersBaseMonthlyReport.vue';
import NabersBaseMeterQuality    from './NabersBaseMeterQuality.vue';

const REFRESH_DAILY_MS = 24 * 60 * 60 * 1000;

const uiConfig = useUiConfigStore();
const store    = useNabersBaseBuildingStore();

const bbCfg  = computed(() => uiConfig.config?.nabersBaseBuilding ?? {});
const enabled = computed(() => bbCfg.value.enabled ?? false);

// Which design stage the dfpTargets were transcribed from. Stage 4 is the usual
// case; a project working from an as-built update overrides this so the chart
// and stat card don't claim a stage the numbers didn't come from.
const dfpTargetLabel = computed(() => bbCfg.value.dfpTargetLabel ?? 'DfP Stage 4 Target');
const dfpTargetSource = computed(() => dfpTargetLabel.value.replace(/ Target$/, ''));

// The tracker takes an ISO date; the store resolves the period start (config
// anchor, else the calendar year) so both read the same window.
const ratingPeriodStartIso = computed(() =>
  bbCfg.value.ratingPeriodStart ?? format(store.ratingPeriodStart, 'yyyy-MM-dd')
);

// Below 0.1% "0.0%" reads as a bug beside a banner saying data was estimated.
const estimatedShareLabel = computed(() => {
  const pct = store.estimatedShare ?? 0;
  return pct < 0.1 ? 'Less than 0.1%' : `${pct.toFixed(1)}%`;
});

// ── Derived display values ────────────────────────────────────────────────────
// Both comparison cards, and the scenario gauge below them, colour from the one
// ladder in util/dfpSeverity.js. They used to grade the same building on three
// separate scales, which put a green card beside two red ones on a building
// holding 5.55 stars with 27% headroom. See that module for the reasoning.
const dfpDiffPct = computed(() => store.dfpDiffPct);

const dfpDiffLabel = computed(() => {
  if (dfpDiffPct.value === null) return null;
  const abs = Math.abs(dfpDiffPct.value).toFixed(1);
  return dfpDiffPct.value <= 0 ? `−${abs}%` : `+${abs}%`;
});

const dfpDiffColor = computed(() => dfpSeverityColor(dfpSeverity({
  intensity:            store.totalIntensity,
  designTarget:         store.dfpTargets.total ?? null,
  headroomPct:          store.headroomPct,
  recommendedMarginPct: store.recommendedMarginPct
})));

const carbonValue = computed(() =>
  store.carbonIntensity !== null ? store.carbonIntensity.toFixed(2) : null
);

// ── Headroom to the target rating ─────────────────────────────────────────────
// Named for what it measures. As "Design margin" it read as a margin against the
// design, which is the one thing it is not: it is measured against the NABERS
// benchmark, so it sat green at 27.3% beside a "vs DfP target" card reading
// +16.3%, and the pair looked contradictory.
const targetStarsLabel = computed(() => {
  const stars = store.targetStars;
  return stars % 1 === 0 ? String(stars) : stars.toFixed(1);
});

const targetStarLabel = computed(() => `${targetStarsLabel.value}★ limit`);

const headroomTitle = computed(() => `Headroom to ${targetStarsLabel.value}★`);

const headroomLabel = computed(() => {
  if (store.headroomPct === null) return null;
  return `${store.headroomPct.toFixed(1)}%`;
});

const headroomColor = computed(() =>
  dfpSeverityColor(headroomSeverity(store.headroomPct, store.recommendedMarginPct))
);

const headroomSubtitle = computed(() => {
  const ceiling = store.targetStarMax;
  const at = ceiling !== null ? ` of ${ceiling.toFixed(1)} kWhe/m²/yr` : '';
  return `Margin below the computed ${targetStarsLabel.value}-star intensity ceiling${at}, ` +
    'against the NABERS benchmark rather than the design target. ' +
    `DfP recommends ≥ ${store.recommendedMarginPct}%`;
});

// ── Headroom to the next rung up ──────────────────────────────────────────────
// The same measurement one published rung above the target, so the pair reads as
// "the rating we hold, and what the next one would cost". Same formula and same
// units as the card beside it, deliberately — the store computes both through
// util/nabersRating.js's headroomPct.
const stretchStarsLabel = computed(() => {
  const stars = store.stretchTarget?.stars ?? null;
  if (stars === null) return null;
  return stars % 1 === 0 ? String(stars) : stars.toFixed(1);
});

// A card still needs a title to explain its own em dash, so there is a wording
// for the case where there is no rung above the target to name.
const stretchTitle = computed(() =>
  stretchStarsLabel.value === null
    ? 'Headroom to next rating'
    : `Headroom to ${stretchStarsLabel.value}★`
);

const stretchLabel = computed(() => {
  const pct = store.stretchHeadroomPct;
  if (pct === null) return null;
  const abs = Math.abs(pct).toFixed(1);
  return pct < 0 ? `−${abs}%` : `+${abs}%`;
});

// Not `headroomSeverity`, which is the one exception to the note above: it turns
// red below the recommended margin, and that is right for a rating the building
// has committed to and could lose. A stretch rung is not a commitment — a
// building comfortably holding 5★ reads about −45% against 5.5★, and painting
// that red beside a green "Headroom to 5★" would tell the reader the opposite of
// the truth. So: green once the rung is within reach, otherwise neutral. Still
// the shared constants, so no fourth colour enters the dashboard.
const stretchColor = computed(() => {
  const pct = store.stretchHeadroomPct;
  if (pct === null) return DFP_SEVERITY_COLOR.unknown;
  return pct >= 0 ? DFP_SEVERITY_COLOR.good : DFP_SEVERITY_COLOR.unknown;
});

const stretchSubtitle = computed(() => {
  const stars = stretchStarsLabel.value;
  const ceiling = store.stretchTarget?.ceiling ?? null;
  if (stars === null || ceiling === null) {
    return 'Margin below the intensity ceiling of the next rating up.';
  }
  const needs = `${stars}★ needs ≤ ${ceiling.toFixed(1)} kWhe/m²/yr`;
  const cut = store.stretchReductionNeeded;
  if (cut === null) return `${needs}.`;
  return cut > 0
    ? `${needs} — a further ${cut.toFixed(1)} kWhe/m²/yr off current use.`
    : `${needs}, which current use already meets.`;
});

// ── Why a stat card has no figure ─────────────────────────────────────────────
// All three cards divide the rating's intensity by something, so they go blank
// together the moment there is no rating — and a bare em dash gives no clue
// whether that is a config omission, a dead board or simply too early in the
// period. The gauge already names its own state; these give the cards the same
// courtesy, in the same order of specificity.
const noRatingReason = computed(() => {
  // Config faults first: knowable with no data at all, and they will not resolve
  // by waiting.
  if (!store.hasConfiguredMeters) return 'No base building meters are configured';
  if (store.missingInputs.length) {
    return `Rating input missing from config: ${store.missingInputs.join(', ')}`;
  }
  // `store.loading` cannot actually reach here — the whole section is a spinner
  // while it is true — but `monthlyLoading` can, and did: the period-to-date read
  // resolves first, so the cards render while the twelve-month table is still in
  // flight and every reason below this line is a claim about absent data. Six
  // days into a period that surfaced as "too few measured months to annualise
  // from", moments before twelve of them arrived.
  if (store.loading || store.monthlyLoading) return 'Loading…';
  // Naming the board is actionable where naming the end use is not.
  if (store.unreadableMeterLabels.length) {
    return `No rating yet: awaiting ${store.unreadableMeterLabels.join('; ')}`;
  }
  if (store.unreadableLabels.length) {
    return `No rating yet: awaiting ${store.unreadableLabels.join(', ')}`;
  }
  // Only when there is no measured basis either. The rating period turning over
  // used to blank every card for four weeks with this message, even though the
  // months behind us were perfectly good to annualise from.
  if (!store.canProject && !store.canUseTrailing) {
    return `${store.elapsedDays} day(s) into the rating period, and too few measured ` +
      'months behind it to annualise from instead';
  }
  return 'No rating yet';
});

const dfpDiffReason = computed(() => {
  if ((store.dfpTargets.total ?? null) === null) return 'No dfpTargets.total in config';
  return noRatingReason.value;
});

const carbonReason = computed(() => {
  if (store.carbonFactor === null) return 'No carbonFactor in config';
  return noRatingReason.value;
});

const headroomReason = computed(() => {
  if (store.targetStarMax === null) {
    return 'Benchmark unavailable; needs a valid postcode and rated hours';
  }
  return noRatingReason.value;
});

const stretchReason = computed(() => {
  // Most specific first, and this one is not a fault to fix: there is no rung
  // above 6★ to have headroom against.
  if (store.stretchTarget === null) return 'Already targeting the top NABERS rating (6★)';
  if (store.stretchTarget.ceiling === null) {
    return 'Benchmark unavailable; needs a valid postcode and rated hours';
  }
  return noRatingReason.value;
});

// For the energy source chart, pass the gross intensity (before PV subtraction)
const grossKwhPerM2 = computed(() => store.grossIntensity);

// ── Lifecycle ─────────────────────────────────────────────────────────────────
let _dailyInterval;

onMounted(() => {
  if (!enabled.value) return;
  store.refresh();
  store.refreshMonthly();
  store.refreshMeterStatuses();
  // Names and refs change when the building is re-commissioned, not daily, so
  // this is fetched once per visit and not on the refresh interval.
  store.refreshMeterMetadata();
  _dailyInterval = setInterval(() => {
    store.refresh();
    store.refreshMonthly();
  }, REFRESH_DAILY_MS);
});

onUnmounted(() => clearInterval(_dailyInterval));
</script>

<style scoped>
.nabers-base-section {
  background:     var(--sc-black, #0C0921);
  padding:        24px 40px 80px 40px;
  font-family:    'Poppins', sans-serif;
  min-height:     100%;
  display:        flex;
  flex-direction: column;
  box-sizing:     border-box;
}

.section-header {
  display:         flex;
  align-items:     center;
  justify-content: space-between;
  flex-shrink:     0;
  margin-bottom:   8px;
}

.section-title {
  font-size:      28px;
  font-weight:    300;
  color:          #ffffff;
  margin:         0;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.nav-link {
  text-decoration: none;
}

.state-wrapper {
  display:         flex;
  justify-content: center;
  align-items:     center;
  flex:            1;
  min-height:      200px;
}

.nabers-body {
  display:        flex;
  flex-direction: column;
  gap:            16px;
  flex:           1;
}

.row {
  display: flex;
  gap:     16px;
}

/* Amber, matching the estimated state everywhere else in the dashboard. */
.estimation-banner {
  display:       flex;
  align-items:   flex-start;
  gap:           10px;
  flex-shrink:   0;
  background:    rgba(247, 161, 44, 0.1);
  border:        1px solid rgba(247, 161, 44, 0.3);
  border-radius: 8px;
  padding:       10px 14px;
  font-size:     13px;
  line-height:   1.45;
  color:         rgba(255, 255, 255, 0.75);
}

.estimation-banner strong {
  color:       #f7a12c;
  font-weight: 500;
}

.estimation-banner .v-icon {
  color:       #f7a12c;
  margin-top:  1px;
  flex-shrink: 0;
}

/* Row 1: the rating, full width, with its supporting detail to its right.
   Carded to match the stat cards below and the widgets further down — as a bare
   full-width strip on the page background it read as adrift rather than as a
   panel. The gauge draws its own padding, so the card is only the frame. */
.row--rating {
  flex-shrink: 0;
}

.row--rating > * {
  flex:          1;
  min-width:     0;
  background:    rgba(255, 255, 255, 0.05);
  border:        1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
}

/* Row 2: 4 stat cards, each `flex: 1` in its own right. */
.row--stats {
  align-items: stretch;
  flex-shrink: 0;
}

/* Row 3: charts */
.row--charts {
  /* Tall enough for the breakdown chart's horizontal mode at eleven end uses. */
  height:      420px;
  flex-shrink: 0;
}

.chart-breakdown {
  flex:      55;
  min-width: 0;
}

.chart-trend {
  flex:      45;
  min-width: 0;
}

/* Row 4: scenario gauge */
.row--gauge {
  flex-shrink: 0;
}

.row--gauge > * {
  flex: 1;
}

/* Row 5: 3 equal bottom cards */
.row--bottom {
  align-items: stretch;
  flex-shrink: 0;
}

/* Accordions */
.accordions {
  margin-top:  8px;
  border:      1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px !important;
  overflow:    hidden;
}

.panel-title {
  font-size:      13px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.6);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

:deep(.v-expansion-panel-title) {
  color: rgba(255,255,255,0.6) !important;
}

:deep(.v-expansion-panel-text__wrapper) {
  padding: 8px 24px 24px;
}
</style>
