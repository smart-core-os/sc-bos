<template>
  <div v-if="enabled" class="nabers-section">
    <div class="section-header">
      <h2 class="section-title">NABERS Tenancy Energy</h2>
      <router-link to="/landlord" class="nav-link">
        <v-btn variant="text" size="small" color="rgba(255,255,255,0.45)">
          Landlord view →
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
      <!-- Row 1: headline stat cards -->
      <div class="row row--cards">
        <NabersStatCard
            title="vs DfP target"
            :value="dfpDiffLabel"
            :value-color="dfpDiffColor"
            :empty-reason="dfpDiffReason"
            :subtitle="dfpDiffSubtitle"/>
        <NabersStatCard
            title="Carbon intensity"
            :value="carbonValue"
            unit="kgCO₂e/m²/yr"
            subtitle="CO₂e from electricity use, per floor area. Reported separately — emission factors do not enter the NABERS rating"/>
        <NabersAfterHoursCard
            :kwh="store.afterHoursKwh"
            :is-after-hours="store.isAfterHours"
            :operating-hours-end="config.nabersOperatingHoursEnd ?? 17"/>
      </div>

      <!-- Row 2: benchmark chart + star rating -->
      <div class="row row--chart">
        <div class="chart-wrapper">
          <NabersBenchmarkChart
              :lighting-actual="store.lightingIntensity"
              :equipment-actual="store.equipmentIntensity"
              :lighting-target="config.nabersBenchmarks?.lighting ?? null"
              :equipment-target="config.nabersBenchmarks?.equipment ?? null"/>
        </div>
        <div class="rating-wrapper">
          <NabersRatingGauge
              boundary-label="Tenancy"
              config-section="nabersNIA / nabersRatedHours"
              :rating="store.headlineRating"
              :is-projection="store.headlineIsProjection"
              :months-of-data="store.monthsOfData"
              :benchmark="store.benchmark"
              :missing-inputs="store.missingInputs"
              :has-meters="store.hasConfiguredMeters"
              :configured-count="configuredStreamCount"
              :unreadable-count="store.unreadableStreams.length"
              :unreadable-labels="store.unreadableStreams"
              :too-early="store.hasConfiguredMeters && !store.canProject && !store.canUseTrailing"
              :loading="store.monthlyLoading"
              :elapsed-days="store.elapsedDays"
              :next-star-target="store.nextStarTarget"
              :reduction-needed="store.reductionNeeded"
              :progress-to-next-star="store.progressToNextStar"
              :estimated-share-pct="store.estimatedShare"
              :estimated-meter-labels="store.estimatedMeterLabels"/>
        </div>
      </div>

      <!-- Row 3: monthly trend -->
      <div class="row row--trend">
        <NabersMonthlyTrend
            :months="store.monthlyData"
            :lighting-target="config.nabersBenchmarks?.lighting ?? null"
            :equipment-target="config.nabersBenchmarks?.equipment ?? null"/>
      </div>

      <!-- Row 4: energy split + scenario gauge -->
      <div class="row row--bottom">
        <NabersDonutChart
            :lighting-kwh="store.lightingIntensity"
            :equipment-kwh="store.equipmentIntensity"/>
        <!-- The tenancy boundary compares against its benchmark, not a DfP
             modelled total, so the reference marker is named for what it is. -->
        <NabersScenarioGauge
            :current-intensity="store.totalIntensity"
            :dfp-total="dfpTotal"
            reference-id="Bench"
            reference-label="Tenancy benchmark total"
            :scenarios="config.nabersScenarios ?? []"/>
      </div>
    </div>
  </div>
</template>

<script setup>
import {computed, onMounted, onUnmounted} from 'vue';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {useNabersMetricsStore} from '@/stores/nabersMetrics.js';
import NabersBenchmarkChart from './NabersBenchmarkChart.vue';
import NabersRatingGauge from './NabersRatingGauge.vue';
import NabersStatCard from './NabersStatCard.vue';
import NabersAfterHoursCard from './NabersAfterHoursCard.vue';
import NabersDonutChart from './NabersDonutChart.vue';
import NabersMonthlyTrend from './NabersMonthlyTrend.vue';
import NabersScenarioGauge from './NabersScenarioGauge.vue';

const REFRESH_DAILY_MS    = 24 * 60 * 60 * 1000;
const REFRESH_15MIN_MS    = 15 * 60 * 1000;

const uiConfig = useUiConfigStore();
const config   = computed(() => uiConfig.config);
const enabled  = computed(() => config.value?.nabersEnabled ?? false);
const store    = useNabersMetricsStore();

// ── Derived display values ────────────────────────────────────────────────────
// The benchmark total is a per-building DfP figure, so there is no default: a
// config without one shows no comparison rather than one against another
// building's target.
const dfpTotal = computed(() => config.value?.nabersBenchmarks?.total ?? null);

const configuredStreamCount = computed(() =>
  [config.value?.nabersLightingMeterNames, config.value?.nabersEquipmentMeterNames]
    .filter(names => (names ?? []).length > 0).length
);

const dfpDiffPct = computed(() => {
  if (store.totalIntensity === null || !dfpTotal.value) return null;
  return ((store.totalIntensity - dfpTotal.value) / dfpTotal.value) * 100;
});

const dfpDiffLabel = computed(() => {
  if (dfpDiffPct.value === null) return null;
  const abs  = Math.abs(dfpDiffPct.value).toFixed(1);
  return dfpDiffPct.value <= 0 ? `−${abs}%` : `+${abs}%`;
});

const dfpDiffColor = computed(() => {
  if (dfpDiffPct.value === null) return '#ffffff';
  return dfpDiffPct.value <= 0 ? '#4caf50' : '#f89c9b';
});

// Naming the configured target beats naming a fixed one, and with no target
// configured the card says so instead of showing a bare dash.
const dfpDiffSubtitle = computed(() =>
  `Annualised total energy use vs the NABERS DfP target of ${dfpTotal.value} kWh/m²/yr`
);

const dfpDiffReason = computed(() => {
  if (dfpTotal.value === null) return 'No nabersBenchmarks.total in config';
  // The monthly table lands after the section has rendered, so "no rating yet"
  // would be asserted about data still in flight.
  if (store.monthlyLoading) return 'Loading…';
  return 'No rating yet';
});

// `?? null`, not a hardcoded fallback: a wrong grid factor silently misreports
// carbon, so an absent one shows as no figure.
const carbonFactor = computed(() => config.value?.nabersCarbonFactor ?? null);

const carbonValue = computed(() => {
  if (store.totalIntensity === null || carbonFactor.value === null) return null;
  return (store.totalIntensity * carbonFactor.value).toFixed(2);
});

// ── Lifecycle ─────────────────────────────────────────────────────────────────
let _dailyInterval;
let _afterHoursInterval;

onMounted(() => {
  if (!enabled.value) return;
  store.refresh();
  store.refreshAfterHours();
  store.refreshMonthly();
  _dailyInterval       = setInterval(() => store.refresh(), REFRESH_DAILY_MS);
  _afterHoursInterval  = setInterval(() => store.refreshAfterHours(), REFRESH_15MIN_MS);
});

onUnmounted(() => {
  clearInterval(_dailyInterval);
  clearInterval(_afterHoursInterval);
});
</script>

<style scoped>
.nabers-section {
  background:     var(--sc-black, #0C0921);
  padding:        24px 40px 120px 40px;
  font-family:    'Poppins', sans-serif;
  height:         100%;
  display:        flex;
  flex-direction: column;
  box-sizing:     border-box;
}

.section-header {
  display:         flex;
  align-items:     center;
  justify-content: space-between;
  margin-bottom:   16px;
  flex-shrink:     0;
}

.nav-link {
  text-decoration: none;
}

.section-title {
  font-size:      28px;
  font-weight:    300;
  color:          #ffffff;
  margin:         0;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.state-wrapper {
  display:         flex;
  justify-content: center;
  align-items:     center;
  flex:            1;
}

.nabers-body {
  display:        flex;
  flex-direction: column;
  gap:            16px;
  flex:           1;
  min-height:     0;
}

.row {
  display: flex;
  gap:     16px;
}

.row--cards {
  align-items: stretch;
  flex-shrink: 0;
}

.row--chart {
  align-items: stretch;
  flex:        3;
  min-height:  0;
}

.row--trend {
  flex:       3;
  min-height: 0;
}

.row--trend > * {
  flex: 1;
}

.chart-wrapper {
  flex:           2;
  min-width:      0;
  display:        flex;
  flex-direction: column;
}

.rating-wrapper {
  flex:            1;
  display:         flex;
  align-items:     center;
  justify-content: center;
}

.row--bottom {
  align-items: stretch;
  flex:        2;
  min-height:  0;
}
</style>
