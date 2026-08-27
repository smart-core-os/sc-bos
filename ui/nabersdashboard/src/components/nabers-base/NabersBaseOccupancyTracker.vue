<template>
  <div class="occupancy-wrapper">
    <div class="occupancy-title">Rating period tracker</div>

    <!-- An existing building has no occupancy gate at all: it is eligible with as
         little as one day's occupancy, and vacancy is handled by shrinking the
         rated area rather than by disqualification. -->
    <template v-if="!isNewBuild">
      <div class="occ-months">
        <span class="occ-months-count">{{ monthsAccrued }}</span>
        <span class="occ-months-label">/ 12 months of data</span>
      </div>

      <v-progress-linear
          :model-value="(monthsAccrued / 12) * 100"
          color="#7F00FF"
          bg-color="rgba(127,0,255,0.18)"
          rounded
          height="10"
          class="occ-progress"/>

      <div class="occ-detail-row">
        <div class="occ-detail">
          <span class="occ-detail-label">Office NLA let</span>
          <span class="occ-detail-value">{{ nlaLetLabel }}</span>
        </div>
        <div class="occ-detail">
          <span class="occ-detail-label">Rating period</span>
          <span class="occ-detail-value">{{ periodStatusLabel }}</span>
        </div>
      </div>

      <div class="occ-desc">
        An existing building can be rated from as little as one day's occupancy — a
        rating needs 12 continuous months of data, and vacancy is handled by
        reducing the rated area, not by an occupancy threshold.
      </div>
    </template>

    <!-- A new build or major refurbishment cannot *start* its rating period until
         the earliest of 75% of office NLA occupied, or two years after the
         occupancy certificate. -->
    <template v-else>
      <div class="occ-months">
        <span class="occ-months-count">{{ startGateMet ? monthsAccrued : '—' }}</span>
        <span class="occ-months-label">
          {{ startGateMet ? '/ 12 months of data' : 'rating period not started' }}
        </span>
      </div>

      <v-progress-linear
          :model-value="startGateMet ? (monthsAccrued / 12) * 100 : 0"
          :color="startGateMet ? '#7F00FF' : 'rgba(255,255,255,0.2)'"
          bg-color="rgba(127,0,255,0.18)"
          rounded
          height="10"
          class="occ-progress"/>

      <div class="occ-detail-row">
        <div class="occ-detail">
          <span class="occ-detail-label">Office NLA let</span>
          <span class="occ-detail-value" :style="{color: nlaColor}">{{ nlaLetLabel }}</span>
        </div>
        <div class="occ-detail">
          <span class="occ-detail-label">Start gate</span>
          <span class="occ-detail-value" :style="{color: startGateMet ? '#4caf50' : '#f7a12c'}">
            {{ startGateLabel }}
          </span>
        </div>
      </div>

      <div class="occ-desc">
        For a new build or major refurbishment the rating period cannot start until
        the earliest of <strong>75% of office NLA occupied</strong> or
        <strong>two years after the occupancy certificate</strong>{{ twoYearSuffix }}.
      </div>
    </template>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {addMonths, addYears, format, differenceInMonths, parseISO, isValid} from 'date-fns';

const NLA_START_THRESHOLD_PCT = 75;
const OCCUPANCY_CERT_FALLBACK_YEARS = 2;

const props = defineProps({
  /** 'newBuild' buildings face the rating-period start gate; existing ones do not. */
  buildingType:             {type: String, default: 'existing'},
  /** Occupied proportion of office net lettable area, %. */
  nlaLetPct:                {type: Number, default: null},
  /** When the 12-month rating period began, ISO date. */
  ratingPeriodStart:        {type: String, default: ''},
  /** Occupancy certificate date, which starts the two-year fallback clock. */
  occupancyCertificateDate: {type: String, default: ''}
});

const isNewBuild = computed(() => props.buildingType === 'newBuild');

const parse = iso => {
  if (!iso) return null;
  const d = parseISO(iso);
  return isValid(d) ? d : null;
};

const periodStart = computed(() => parse(props.ratingPeriodStart));
const certDate    = computed(() => parse(props.occupancyCertificateDate));

/** Months of the 12-month period accrued so far, capped at 12. */
const monthsAccrued = computed(() => {
  if (!periodStart.value) return 0;
  return Math.min(12, Math.max(0, differenceInMonths(new Date(), periodStart.value)));
});

const nlaLetLabel = computed(() =>
  props.nlaLetPct !== null ? `${props.nlaLetPct}%` : '—'
);

const nlaColor = computed(() => {
  if (props.nlaLetPct === null) return 'rgba(255,255,255,0.4)';
  return props.nlaLetPct >= NLA_START_THRESHOLD_PCT ? '#4caf50' : '#f7a12c';
});

/** The date the two-year-since-occupancy-certificate trigger fires. */
const twoYearTrigger = computed(() =>
  certDate.value ? addYears(certDate.value, OCCUPANCY_CERT_FALLBACK_YEARS) : null
);

const nlaThresholdMet = computed(() =>
  props.nlaLetPct !== null && props.nlaLetPct >= NLA_START_THRESHOLD_PCT
);

const twoYearTriggerMet = computed(() =>
  twoYearTrigger.value !== null && twoYearTrigger.value <= new Date()
);

const startGateMet = computed(() => nlaThresholdMet.value || twoYearTriggerMet.value);

const startGateLabel = computed(() => {
  if (nlaThresholdMet.value) return 'Met — NLA ≥ 75%';
  if (twoYearTriggerMet.value) return 'Met — 2 yrs from cert.';
  if (twoYearTrigger.value) return `By ${format(twoYearTrigger.value, 'MMM yyyy')}`;
  if (props.nlaLetPct === null) return 'Set NLA let %';
  return 'Not yet met';
});

/** Names the fallback date in the description when one is configured. */
const twoYearSuffix = computed(() =>
  twoYearTrigger.value ? ` (${format(twoYearTrigger.value, 'd MMM yyyy')})` : ''
);

const periodStatusLabel = computed(() => {
  if (!periodStart.value) return 'Not started';
  if (monthsAccrued.value >= 12) return 'Complete';
  return `From ${format(periodStart.value, 'MMM yyyy')}`;
});

/** Unused by the template but kept for parity with the previous API surface. */
const eligibleFrom = computed(() =>
  periodStart.value ? addMonths(periodStart.value, 12) : null
);
defineExpose({eligibleFrom});
</script>

<style scoped>
.occupancy-wrapper {
  background:     rgba(255, 255, 255, 0.05);
  border:         1px solid rgba(255, 255, 255, 0.08);
  border-radius:  12px;
  padding:        20px 24px;
  display:        flex;
  flex-direction: column;
  gap:            12px;
  flex:           1;
}

.occupancy-title {
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.occ-months {
  display:     flex;
  align-items: baseline;
  gap:         6px;
}

.occ-months-count {
  font-size:   52px;
  font-weight: 300;
  color:       #ffffff;
  line-height: 1;
}

.occ-months-label {
  font-size: 15px;
  color:     rgba(255, 255, 255, 0.45);
}

.occ-progress {
  margin: 4px 0;
}

.occ-detail-row {
  display: flex;
  gap:     24px;
}

.occ-detail {
  display:        flex;
  flex-direction: column;
  gap:            4px;
}

.occ-detail-label {
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.4);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.occ-detail-value {
  font-size:   20px;
  font-weight: 300;
  color:       #ffffff;
}

.occ-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  auto;
  line-height: 1.4;
}

.occ-desc strong {
  color: rgba(255, 255, 255, 0.5);
}
</style>
