<template>
  <div :class="['floor-comfort', { 'floor-comfort--compact': props.compact }]">
    <div :class="['column-headers', { 'column-headers--compact': props.compact }]">
      <span class="header-device">{{ props.compact ? '' : 'Device' }}</span>
      <span
          v-for="col in visibleColumns"
          :key="col.key"
          class="header-label">
        {{ col.label }}
      </span>
    </div>
    <floor-list
        :class="['floor-list-wrap', { 'floor-list-wrap--compact': props.compact }]"
        :floors="props.floors"
        :compact="props.compact">
      <template #floor="{floor}">
        <div :class="['comfort-row', { 'comfort-row--compact': props.compact }]">
          <div :class="['device-col', { 'device-col--compact': props.compact, 'device-col--named': props.showFloorName }]">
            <span v-if="props.showFloorName" class="floor-name-label">{{ floor.title ?? `L${floor.level}` }}</span>
            <device-cell
                v-if="deviceItemByLevel[floor.level]"
                class="device-cell"
                :item="deviceItemByLevel[floor.level]"/>
            <span v-else class="device-cell"/>
          </div>
          <v-tooltip
              v-for="col in visibleColumns"
              :key="col.key"
              location="top"
              :text="col.tooltip(floorData[floor.zoneName])">
            <template #activator="{props: tp}">
              <span
                  v-bind="tp"
                  :class="['comfort-chip', col.colorClass(floorData[floor.zoneName]), { 'comfort-chip--compact': props.compact }]">
                {{ col.display(floorData[floor.zoneName]) }}
              </span>
            </template>
          </v-tooltip>
        </div>
      </template>
    </floor-list>
  </div>
</template>

<script setup>
/**
 * Per-floor view combining:
 *  1. DeviceCell — live trait data (occupancy, temperature, electric, etc.) from device metadata
 *  2. Environmental comfort matrix — colour-coded AQ chips (Temp | RH% | CO₂ | VOC | dB)
 *
 * Props:
 *   floors: [{level, zoneName, title?}]  — same shape as BuildingFloors/FloorList
 *   columns: subset of ['temp','humidity','co2','voc','pm25','sound']
 *   tempSetpoint: comfort target in °C (default 21)
 */
import FloorList from '@/components/FloorList.vue';
import {HOUR, useNow} from '@/components/now.js';
import {useDevicesCollection} from '@/composables/devices.js';
import DeviceCell from '@/routes/devices/components/DeviceCell.vue';
import {useAirQualityPeriodAverages} from '@/dynamic/widgets/environmental/airQuality.js';
import {useAirTemperaturePeriodAverages} from '@/dynamic/widgets/environmental/airTemperature.js';
import {useSoundLevelPeriodAverages} from '@/dynamic/widgets/environmental/soundSensor.js';
import {useAirQuality, usePullAirQuality} from '@/traits/airQuality/airQuality.js';
import {useAirTemperature, usePullAirTemperature} from '@/traits/airTemperature/airTemperature.js';
import {usePullSoundLevel, useSoundLevel} from '@/traits/sound/sound.js';
import {useRollingValue} from '@/composables/rollingValue.js';
import {computed, effectScope, onScopeDispose, reactive, watch} from 'vue';

const props = defineProps({
  floors: {
    type: Array, // [{level: number, zoneName: string, title?: string}]
    required: true,
  },
  columns: {
    type: Array, // subset of ['temp', 'humidity', 'co2', 'voc', 'pm25', 'sound']
    default: () => ['temp', 'humidity', 'co2', 'voc', 'pm25', 'sound'],
  },
  tempSetpoint: {
    type: Number,
    default: 21,
  },
  // When true, show ↑↓ trend arrows in chips comparing current live values to the prior 24 h.
  showTrend: {
    type: Boolean,
    default: false,
  },
  // When true, renders each floor as a compact fixed-height row with scrolling and a sticky header.
  // Useful when there are many floors (20+) that would otherwise be squished.
  compact: {
    type: Boolean,
    default: false,
  },
  // When true, shows the floor title (or "L{level}" fallback) in the device column.
  showFloorName: {
    type: Boolean,
    default: true,
  },
});

// ── Device metadata + DeviceCell per floor ────────────────────────────────────

const devicesReq = computed(() => ({
  query: {
    conditionsList: [
      {field: 'name', stringIn: {stringsList: props.floors.map(f => f.zoneName).filter(Boolean)}},
    ],
  },
}));
const deviceCollection = useDevicesCollection(devicesReq);

const floorByZoneName = computed(() => {
  const res = {};
  for (const floor of props.floors) {
    if (floor.zoneName) res[floor.zoneName] = floor;
  }
  return res;
});

const deviceItemByLevel = computed(() => {
  const byZoneName = floorByZoneName.value;
  const dst = {};
  for (const item of deviceCollection.items.value) {
    const floor = byZoneName[item.name];
    if (floor) dst[floor.level] = item;
  }
  return dst;
});

// ── AQ comfort matrix: per-zone sensor data ───────────────────────────────────

const floorData = reactive(/** @type {Record<string, {temp, humidity, co2, voc, pm25, sound, trends?}>} */ {});
const scopeByZone = {};

// Prior-period edges (prior 24 h window), refreshed every hour.
const {now} = useNow(HOUR);
const priorStart = computed(() => new Date(now.value - 48 * 3600_000));
const priorEnd = computed(() => new Date(now.value - 24 * 3600_000));
const AQ_TREND_METRICS = ['carbonDioxideLevel', 'volatileOrganicCompounds', 'particulateMatter25'];
const AT_TREND_METRICS = ['ambientTemperature', 'ambientHumidity'];
const SL_TREND_METRICS = ['soundPressureLevel'];

watch(() => props.floors.map(f => f.zoneName), (zoneNames) => {
  const toStop = new Set(Object.keys(scopeByZone));

  for (const zoneName of zoneNames) {
    if (!zoneName) continue;
    if (scopeByZone[zoneName]) {
      toStop.delete(zoneName);
      continue;
    }
    const scope = effectScope();
    scopeByZone[zoneName] = scope;
    scope.run(() => {
      const {value: aq} = usePullAirQuality(zoneName);
      const {presentMetrics} = useAirQuality(aq);

      const {value: at} = usePullAirTemperature(zoneName);
      const {temp, humidity} = useAirTemperature(at);

      const {value: sl} = usePullSoundLevel(zoneName);
      const {soundPressureLevel} = useSoundLevel(sl);

      // Prior-period averages for trend arrows — only fetched when showTrend is on
      const aqAvgs = useAirQualityPeriodAverages(
        computed(() => props.showTrend ? zoneName : null),
        AQ_TREND_METRICS,
        priorStart,
        priorEnd
      );
      const atAvgs = useAirTemperaturePeriodAverages(
        computed(() => props.showTrend ? zoneName : null),
        AT_TREND_METRICS,
        priorStart,
        priorEnd
      );
      const slAvgs = useSoundLevelPeriodAverages(
        computed(() => props.showTrend ? zoneName : null),
        SL_TREND_METRICS,
        priorStart,
        priorEnd
      );

      const trends = {
        co2: useRollingValue(computed(() => presentMetrics.value?.carbonDioxideLevel?.value), computed(() => aqAvgs.value.carbonDioxideLevel)),
        voc: useRollingValue(computed(() => presentMetrics.value?.volatileOrganicCompounds?.value), computed(() => aqAvgs.value.volatileOrganicCompounds)),
        pm25: useRollingValue(computed(() => presentMetrics.value?.particulateMatter25?.value), computed(() => aqAvgs.value.particulateMatter25)),
        temp: useRollingValue(temp, computed(() => atAvgs.value.ambientTemperature)),
        humidity: useRollingValue(humidity, computed(() => atAvgs.value.ambientHumidity)),
        sound: useRollingValue(soundPressureLevel, computed(() => slAvgs.value.soundPressureLevel)),
      };

      watch([presentMetrics, temp, humidity, soundPressureLevel, aqAvgs, atAvgs, slAvgs], ([pm, t, h, sl]) => {
        floorData[zoneName] = {
          temp: t ?? null,
          humidity: h ?? null,
          co2: pm?.carbonDioxideLevel?.value ?? null,
          voc: pm?.volatileOrganicCompounds?.value ?? null,
          pm25: pm?.particulateMatter25?.value ?? null,
          sound: sl ?? null,
          trends,
        };
      }, {immediate: true, deep: true});
    });
  }

  for (const zoneName of toStop) {
    scopeByZone[zoneName].stop();
    delete scopeByZone[zoneName];
    delete floorData[zoneName];
  }
}, {immediate: true});

onScopeDispose(() => {
  for (const scope of Object.values(scopeByZone)) scope.stop();
});

// Returns ↑ / ↓ / '' based on % change from prior to current
/**
 * Returns an arrow indicator based on the percentage change between current and prior values.
 *
 * @param {object|null} rolling - Result from useRollingValue
 * @return {string} ' ↑', ' ↓', or ''
 */
function trendArrow(rolling) {
  if (!props.showTrend || !rolling || rolling.percentChange == null) return '';
  if (Math.abs(rolling.percentChange) < 0.05) return ''; // < 0.05% change — ignore noise
  return rolling.percentChange > 0 ? ' ↑' : ' ↓';
}
/**
 * Generates a tooltip suffix with the prior value and unit if trend display is enabled.
 *
 * @param {object|null} rolling - Result from useRollingValue
 * @param {string} [unit] - The unit to display
 * @return {string} Tooltip suffix or empty string
 */
function trendTooltipSuffix(rolling, unit = '') {
  if (!props.showTrend || !rolling || rolling.baselineValue == null) return '';
  const prior = rolling.baselineValue;
  const val = typeof prior === 'number' ? prior.toFixed(prior < 10 ? 2 : 0) : '—';
  return ` (prior: ${val}${unit ? ' ' + unit : ''})`;
}

// Column definitions — computed so display/tooltip close over reactive props.showTrend
const allColumns = computed(() => [
  {
    key: 'temp',
    label: 'Temp',
    display: (d) => {
      if (d?.temp == null) return '—';
      return `${d.temp.toFixed(1)}°${trendArrow(d.trends?.temp)}`;
    },
    tooltip: (d) => d?.temp != null
      ? `Temperature: ${d.temp.toFixed(1)} °C${trendTooltipSuffix(d.trends?.temp, '°C')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.temp === null) return 'chip--unknown';
      const dev = Math.abs(d.temp - props.tempSetpoint);
      if (dev <= 2) return 'chip--good';
      if (dev <= 4) return 'chip--warn';
      return 'chip--bad';
    },
  },
  {
    key: 'humidity',
    label: 'RH%',
    display: (d) => {
      if (d?.humidity == null) return '—';
      return `${d.humidity.toFixed(0)}%${trendArrow(d.trends?.humidity)}`;
    },
    tooltip: (d) => d?.humidity != null
      ? `Humidity: ${d.humidity.toFixed(1)} %${trendTooltipSuffix(d.trends?.humidity, '%')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.humidity === null) return 'chip--unknown';
      const h = d.humidity;
      if (h >= 40 && h <= 60) return 'chip--good';
      if (h >= 30 && h <= 70) return 'chip--warn';
      return 'chip--bad';
    },
  },
  {
    key: 'co2',
    label: 'CO₂',
    display: (d) => {
      if (d?.co2 == null) return '—';
      return `${Math.round(d.co2)}${trendArrow(d.trends?.co2)}`;
    },
    tooltip: (d) => d?.co2 != null
      ? `CO₂: ${Math.round(d.co2)} ppm${trendTooltipSuffix(d.trends?.co2, 'ppm')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.co2 === null) return 'chip--unknown';
      if (d.co2 < 1000) return 'chip--good';
      if (d.co2 < 2000) return 'chip--warn';
      return 'chip--bad';
    },
  },
  {
    key: 'voc',
    label: 'VOC',
    display: (d) => {
      if (d?.voc == null) return '—';
      return `${d.voc.toFixed(2)}${trendArrow(d.trends?.voc)}`;
    },
    tooltip: (d) => d?.voc != null
      ? `VOC: ${d.voc.toFixed(2)} ppm${trendTooltipSuffix(d.trends?.voc, 'ppm')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.voc === null) return 'chip--unknown';
      if (d.voc < 0.3) return 'chip--good';
      if (d.voc < 0.5) return 'chip--warn';
      return 'chip--bad';
    },
  },
  {
    key: 'pm25',
    label: 'PM2.5',
    display: (d) => {
      if (d?.pm25 == null) return '—';
      return `${Math.round(d.pm25)}${trendArrow(d.trends?.pm25)}`;
    },
    tooltip: (d) => d?.pm25 != null
      ? `PM2.5: ${Math.round(d.pm25)} µg/m³${trendTooltipSuffix(d.trends?.pm25, 'µg/m³')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.pm25 === null) return 'chip--unknown';
      if (d.pm25 < 10) return 'chip--good';
      if (d.pm25 < 20) return 'chip--warn';
      return 'chip--bad';
    },
  },
  {
    key: 'sound',
    label: 'dB',
    display: (d) => {
      if (d?.sound == null) return '—';
      return `${d.sound.toFixed(0)}${trendArrow(d.trends?.sound)}`;
    },
    tooltip: (d) => d?.sound != null
      ? `Sound: ${d.sound.toFixed(1)} dB${trendTooltipSuffix(d.trends?.sound, 'dB')}`
      : 'No data',
    colorClass: (d) => {
      if (!d || d.sound === null) return 'chip--unknown';
      if (d.sound < 50) return 'chip--good';
      if (d.sound < 65) return 'chip--warn';
      return 'chip--bad';
    },
  },
]);

const visibleColumns = computed(() =>
  allColumns.value.filter(c => props.columns.includes(c.key))
);


</script>

<style scoped>
.floor-comfort {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.floor-comfort--compact {
}

.column-headers {
  display: grid;
  grid-template-columns: 1fr repeat(v-bind('visibleColumns.length'), minmax(0, 2.5rem));
  gap: 3px;
  align-items: center;
  padding-bottom: 8px;
  margin-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
  flex-shrink: 0;
}

.header-device {
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.header-label {
  display: flex;
  align-items: end;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.floor-list-wrap {
  height: 100%;
}

.floor-list-wrap--compact {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.floor-comfort :deep(.floor-list) {
  min-height: 0;
}

.comfort-row {
  display: grid;
  grid-template-columns: 1fr repeat(v-bind('visibleColumns.length'), minmax(0, 2.5rem));
  gap: 3px;
  align-items: center;
}


.comfort-row--compact {
  gap: 2px;
  height: 20px;
  overflow: hidden;
}

.comfort-chip--compact {
  height: 18px;
  font-size: 0.6rem;
}

.device-col {
  display: contents; /* non-compact: DeviceCell is a direct grid child */
}

.device-col--named {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 4px;
  overflow: hidden;
  min-width: 0;
}

.device-col--named .device-cell {
  flex: 1;
  min-width: 0;
}

.floor-name-label {
  font-size: 0.7rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 1;
  min-width: 0;
}

.device-col--compact {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 4px;
  overflow: hidden;
  min-width: 0;
}

/* Resize DeviceCell sub-components to fit compact 20px rows */
.device-col--compact :deep(.v-icon) {
  font-size: 14px;
}
.device-col--compact :deep(.v-avatar) {
  width: 16px !important;
  height: 16px !important;
}
.device-col--compact :deep(.overlap) {
  --size: 16px;
}

.device-cell {
  overflow: hidden;
  min-width: 0;
}


.comfort-chip {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 500;
  height: 22px;
  cursor: default;
  user-select: none;
}

.chip--good {
  background: rgba(76, 175, 80, 0.30);
  color: #a5d6a7;
}

.chip--warn {
  background: rgba(255, 152, 0, 0.30);
  color: #ffcc80;
}

.chip--bad {
  background: rgba(244, 67, 54, 0.30);
  color: #ef9a9a;
}

.chip--unknown {
  background: rgba(255, 255, 255, 0.07);
  color: rgba(255, 255, 255, 0.35);
}
</style>
