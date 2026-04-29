<template>
  <v-card :loading="anyLoading" class="lift-card d-flex flex-column" :style="{ height: formatHeight(props.height) }">
    <v-toolbar v-if="!props.hideToolbar" color="transparent" density="comfortable">
      <v-toolbar-title class="text-h6 font-weight-bold">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="flex-grow-1 overflow-hidden pt-0">
      <div :class="['lift-grid', { 'lift-grid--compact': props.compact }]">
        <!-- Shaft names header — stays fixed while floors scroll in compact mode -->
        <div class="lift-header">
          <div class="floor-label-spacer"/>
          <div class="shafts-names-row">
            <div v-for="name in deviceNames" :key="name" class="shaft-name" :title="name">
              {{ shortName(name) }}
            </div>
          </div>
        </div>
        <!-- Floor grid — scrollable in compact mode -->
        <div class="lift-body">
          <div class="floor-labels">
            <div v-for="floorNum in floors" :key="floorNum" class="floor-label">
              {{ floorNum }}
            </div>
          </div>
          <div class="shafts-tracks">
            <div v-for="name in deviceNames" :key="name" class="shaft-track">
              <div v-for="floorNum in floors" :key="floorNum" class="floor-cell">
                <div v-if="isAtFloor(name, floorNum)"
                     class="lift-car"
                     :class="directionClass(name)">
                  <v-icon :icon="directionIcon(name)" class="lift-icon"/>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import { usePullTransport, useTransport } from '@/traits/transport/transport.js';
import { computed, effectScope, onScopeDispose, reactive, watch } from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: 'Lift Positions'
  },
  deviceNames: {
    type: Array,
    default: () => []
  },
  floorCount: {
    type: Number,
    default: 0
  },
  floors: {
    type: Array,
    default: () => []
  },
  hideToolbar: {
    type: Boolean,
    default: false
  },
  height: {
    type: [String, Number],
    default: '100%'
  },
  // When true, renders each floor as a compact fixed-height row with scrolling.
  // Useful when there are many floors (20+) that would otherwise be squished.
  compact: {
    type: Boolean,
    default: false
  }
});

const floors = computed(() => {
  if (props.floors != null && props.floors.length > 0) {
    return props.floors;
  }
  const res = [];
  // Standard lift grid from top to bottom, including ground floor.
  for (let i = props.floorCount; i >= 1; i--) {
    res.push(i.toString());
  }
  res.push('GF');
  return res;
});

const liftData = reactive({});
const scopeByDevice = {};

watch(() => props.deviceNames, (names) => {
  const toStop = new Set(Object.keys(scopeByDevice));

  for (const name of names) {
    if (!name) continue;
    if (scopeByDevice[name]) {
      toStop.delete(name);
      continue;
    }
    const scope = effectScope();
    scopeByDevice[name] = scope;
    scope.run(() => {
      const { value, loading } = usePullTransport(name);
      const { actualPosition, movingDirection, doorStatus } = useTransport(value);

      watch([actualPosition, movingDirection, doorStatus, loading], ([pos, dir, doors, l]) => {
        liftData[name] = {
          pos,
          dir,
          doors,
          loading: l
        };
      }, { immediate: true });
    });
  }

  for (const name of toStop) {
    scopeByDevice[name].stop();
    delete scopeByDevice[name];
    delete liftData[name];
  }
}, { immediate: true });

onScopeDispose(() => {
  for (const scope of Object.values(scopeByDevice)) {
    scope.stop();
  }
});

const anyLoading = computed(() => Object.values(liftData).some(d => d.loading));

/**
 *
 * @param name
 * @param floorNum
 */
function isAtFloor(name, floorNum) {
  const currentPos = liftData[name]?.pos;
  if (!currentPos) return false;
  // Handle if floorNum is a string and currentPos is a string/number
  // Trim spaces to handle mock driver formatting like " 1"
  return currentPos.toString().trim() === floorNum.toString().trim();
}

/**
 *
 * @param name
 */
function directionIcon(name) {
  const data = liftData[name];
  const dir = data?.dir;
  if (dir && dir.includes('UP')) return 'mdi-chevron-up';
  if (dir && dir.includes('DOWN')) return 'mdi-chevron-down';

  if (isDoorOpen(name)) return 'mdi-door-open';
  return 'mdi-elevator';
}

/**
 *
 * @param name
 */
function isDoorOpen(name) {
  const data = liftData[name];
  if (!data || !data.doors) return false;
  return data.doors.some(d => d.value === 'Open' || d.value === 'Opening');
}

/**
 *
 * @param name
 */
function directionClass(name) {
  const data = liftData[name];
  const dir = data?.dir;
  if (dir && dir.includes('UP')) return 'lift-moving-up';
  if (dir && dir.includes('DOWN')) return 'lift-moving-down';
  if (isDoorOpen(name)) return 'lift-door-open';
  return '';
}

/**
 *
 * @param h
 */
function formatHeight(h) {
  if (typeof h === 'number') return `${h}px`;
  return h;
}

/**
 *
 * @param name
 */
function shortName(name) {
  return name.split('/').pop();
}
</script>

<style scoped>
/* ── Outer grid: column layout so the header row sits above the floor body ─── */
.lift-grid {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* ── Header: shaft names + floor-label column spacer ─────────────────────── */
.lift-header {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
  align-items: flex-end;
  padding-bottom: 4px;
}

/* Same width as .floor-label so shaft names align with shaft tracks */
.floor-label-spacer {
  width: 24px;
  flex-shrink: 0;
}

.shafts-names-row {
  display: flex;
  gap: 12px;
  flex-grow: 1;
  min-width: 0;
}

.shaft-name {
  flex: 1;
  min-width: 32px;
  height: 20px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-align: center;
}

/* ── Body: floor labels + shaft tracks ───────────────────────────────────── */
.lift-body {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding-bottom: 8px;
}

.floor-labels {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.floor-label {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  width: 24px;
}

.shafts-tracks {
  display: flex;
  gap: 12px;
  flex-grow: 1;
  min-width: 0;
}

.shaft-track {
  flex: 1;
  min-width: 32px;
  min-height: 0;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.floor-cell {
  flex: 1;
  min-height: 0;
  width: 100%;
  border-bottom: 1px dashed rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
}

.floor-cell:last-child {
  border-bottom: none;
}

/* ── Compact mode ─────────────────────────────────────────────────────────── */
/*
 * In compact mode, floor rows have a fixed height instead of flex-stretching.
 * .floor-labels and .shafts-tracks use align-self: flex-start so their natural
 * content height (floors × 20px) causes .lift-body to overflow and scroll,
 * rather than being stretched down to the body's height.
 *
 * Shaft gap is reduced from 12px → 4px and min-width from 32px → 20px so that
 * many shafts (e.g. 32) fit side-by-side without overflow or clipping.
 */
.lift-grid--compact .lift-body {
  overflow: hidden auto; /* clip x, scroll y */
}

.lift-grid--compact .floor-labels,
.lift-grid--compact .shafts-tracks {
  align-self: flex-start;
}

/* Tighter gap between shafts — must match on both the header names row and the
   tracks row so shaft names stay aligned above their tracks. */
.lift-grid--compact .shafts-names-row,
.lift-grid--compact .shafts-tracks {
  gap: 4px;
}

.lift-grid--compact .shaft-name,
.lift-grid--compact .shaft-track {
  min-width: 20px;
}

.lift-grid--compact .floor-label {
  flex: 0 0 20px;
}

.lift-grid--compact .floor-cell {
  flex: 0 0 20px;
}

.lift-grid--compact .lift-car {
  width: 18px;
  height: 18px;
}

.lift-grid--compact .lift-icon {
  font-size: 0.8rem;
}

/* ── Lift car states ──────────────────────────────────────────────────────── */
.lift-car {
  background: rgb(var(--v-theme-primary));
  color: white;
  border-radius: 4px;
  width: 34px;
  height: 34px;
  max-width: 90%;
  max-height: 90%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.lift-icon {
  font-size: 1.1rem;
  width: 100%;
  height: 100%;
  max-width: 90%;
  max-height: 90%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.lift-moving-up {
  background: #4caf50 !important;
}

.lift-moving-down {
  background: #2196f3 !important;
}

.lift-door-open {
  background: #ff9800 !important;
}
</style>
