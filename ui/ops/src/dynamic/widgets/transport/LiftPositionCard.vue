<template>
  <v-card :loading="anyLoading" class="lift-card d-flex flex-column" :style="{ height: formatHeight(props.height) }">
    <v-toolbar v-if="!props.hideToolbar" color="transparent" density="comfortable">
      <v-toolbar-title class="text-h6 font-weight-bold">{{ props.title }}</v-toolbar-title>
    </v-toolbar>
    <v-card-text class="flex-grow-1 overflow-auto pt-0">
      <div class="lift-grid">
        <div class="floor-labels">
          <div v-for="floorNum in floors" :key="floorNum" class="floor-label">
            {{ floorNum }}
          </div>
        </div>
        <div class="shafts-container">
          <div v-for="name in deviceNames" :key="name" class="shaft">
            <div class="shaft-name text-caption text-center" :title="name">
              {{ shortName(name) }}
            </div>
            <div class="shaft-track">
              <div v-for="floorNum in floors" :key="floorNum" class="floor-cell">
                <div v-if="isAtFloor(name, floorNum)"
                     class="lift-car"
                     :class="directionClass(name)">
                  <v-icon :icon="directionIcon(name)" size="small"/>
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

function isAtFloor(name, floorNum) {
  const currentPos = liftData[name]?.pos;
  if (!currentPos) return false;
  // Handle if floorNum is a string and currentPos is a string/number
  // Trim spaces to handle mock driver formatting like " 1"
  return currentPos.toString().trim() === floorNum.toString().trim();
}

function directionIcon(name) {
  const data = liftData[name];
  const dir = data?.dir;
  if (dir && dir.includes('UP')) return 'mdi-chevron-up';
  if (dir && dir.includes('DOWN')) return 'mdi-chevron-down';

  if (isDoorOpen(name)) return 'mdi-door-open';
  return 'mdi-elevator';
}

function isDoorOpen(name) {
  const data = liftData[name];
  if (!data || !data.doors) return false;
  return data.doors.some(d => d.value === 'Open' || d.value === 'Opening');
}

function directionClass(name) {
  const data = liftData[name];
  const dir = data?.dir;
  if (dir && dir.includes('UP')) return 'lift-moving-up';
  if (dir && dir.includes('DOWN')) return 'lift-moving-down';
  if (isDoorOpen(name)) return 'lift-door-open';
  return '';
}

function formatHeight(h) {
  if (typeof h === 'number') return `${h}px`;
  return h;
}

function shortName(name) {
  return name.split('/').pop();
}
</script>

<style scoped>
.lift-grid {
  display: flex;
  gap: 12px;
  overflow-x: auto;
  padding-bottom: 8px;
  min-height: 100%;
}

.floor-labels {
  display: flex;
  flex-direction: column;
  padding-top: 24px; /* Align with shaft track */
}

.floor-label {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  width: 24px;
}

.shafts-container {
  display: flex;
  gap: 12px;
  flex-grow: 1;
}

.shaft {
  display: flex;
  flex-direction: column;
  min-width: 44px;
}

.shaft-name {
  height: 24px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.shaft-track {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  display: flex;
  flex-direction: column;
}

.floor-cell {
  height: 36px;
  width: 44px;
  border-bottom: 1px dashed rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
}

.floor-cell:last-child {
  border-bottom: none;
}

.lift-car {
  background: rgb(var(--v-theme-primary));
  color: white;
  border-radius: 4px;
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.lift-moving-up {
  background: #4caf50 !important;
}

.lift-moving-down {
  background: #2196f3 !important;
}

.lift-door-open {
  background: #ff9800 !important; /* Orange for door open/opening */
}

</style>
