<template>
  <v-card width="300px" class="ma-2 node-card">
    <div class="node-card__accent" :style="accentStyle"/>
    <v-card-text class="pa-3">
      <!-- Header -->
      <div class="d-flex align-start">
        <div class="flex-grow-1 min-width-0">
          <div class="d-flex align-center flex-wrap" style="gap: 6px">
            <span class="text-subtitle-2 font-weight-bold text-truncate" :title="node.name">
              {{ node.name }}
            </span>
            <!-- Split chip: connected + hub/gateway -->
            <span v-if="node.isServer && isCentralHub"
                  class="split-chip"
                  v-tooltip:bottom="'Connected hub'">
              <span class="split-chip__half split-chip__half--success">connected</span>
              <span class="split-chip__half split-chip__half--orange">hub</span>
            </span>
            <span v-else-if="node.isServer && node.role === NodeRole.GATEWAY"
                  class="split-chip"
                  v-tooltip:bottom="'Connected gateway'">
              <span class="split-chip__half split-chip__half--success">connected</span>
              <span class="split-chip__half split-chip__half--secondary">gateway</span>
            </span>
            <!-- Single chips -->
            <template v-else>
              <v-chip v-if="node.isServer" color="success" size="x-small" variant="flat"
                      v-tooltip:bottom="'The component you are connected to'">
                connected
              </v-chip>
              <v-chip v-if="node.role === NodeRole.GATEWAY" color="secondary" size="x-small" variant="flat">
                gateway
              </v-chip>
              <v-chip v-if="isCentralHub" color="orange" size="x-small" variant="flat">hub</v-chip>
              <v-chip v-if="isProxyHubNode" color="cyan-darken-1" size="x-small" variant="flat">hub-proxy</v-chip>
            </template>
          </div>
          <div class="text-caption text-medium-emphasis d-flex align-center mt-1">
            <v-icon size="11" class="mr-1">mdi-network</v-icon>
            {{ node.grpcAddress }}
          </div>
        </div>
        <v-menu v-if="node.role !== NodeRole.INDEPENDENT" min-width="175px">
          <template #activator="{ props: _props }">
            <v-btn icon="mdi-dots-vertical" variant="text" size="x-small" density="compact" v-bind="_props">
              <v-icon size="16"/>
            </v-btn>
          </template>
          <v-list class="py-0">
            <v-list-item link @click="onShowCertificates(node.grpcAddress)">
              <v-list-item-title>View Certificate</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="hasLogService" link @click="onDownloadLogs(node.name)">
              <v-list-item-title>Download Logs</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="hasLogService" link @click="onViewLiveLogs(node.name)">
              <v-list-item-title>View Live Logs</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="node.role !== NodeRole.HUB && !node.isServer"
                         link
                         @click="onForgetNode(node.grpcAddress)">
              <v-list-item-title class="text-error">Forget Node</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>
      </div>

      <v-divider class="mt-2 mb-3"/>

      <!-- Service stats -->
      <div class="stat-grid">
        <div
            v-for="(response, service) in nodeDetails"
            :key="service"
            class="stat-box"
            :class="{'stat-box--error': response.streamError}">
          <div class="stat-value">
            <template v-if="response.streamError">
              <v-icon size="14" color="error">mdi-alert-circle-outline</v-icon>
            </template>
            <template v-else>
              {{ response.value?.totalActiveCount ?? '—' }}
            </template>
          </div>
          <div class="stat-label">{{ service }}</div>
        </div>
      </div>

      <!-- Resource usage -->
      <with-resource-use :name="node.name" :paused="false" v-slot="{ resource: ruResource }">
        <template v-if="!ruResource.streamError && ruResource.value">
          <v-divider class="my-3"/>
          <div class="resource-use">
            <template v-if="ruResource.value.cpu?.utilization != null">
              <div class="resource-section-label d-flex align-center">
                <span class="flex-grow-1">CPU</span>
                <button
                    v-if="ruResource.value.cpu.corePercentList?.length"
                    class="expand-btn"
                    @click="cpuExpanded = !cpuExpanded">
                  {{ cpuExpanded ? '−' : '+' }}
                </button>
              </div>
              <div class="resource-row">
                <span class="resource-label">Average</span>
                <v-progress-linear
                    :model-value="ruResource.value.cpu.utilization"
                    :color="utilizationColor(ruResource.value.cpu.utilization)"
                    height="3"
                    rounded
                    class="resource-bar"/>
                <span class="resource-value">{{ ruResource.value.cpu.utilization.toFixed(1) }}%</span>
              </div>
              <v-expand-transition>
                <div v-show="cpuExpanded">
                  <div
                      v-for="(core, i) in ruResource.value.cpu.corePercentList"
                      :key="i"
                      class="resource-row resource-row--core">
                    <span class="resource-label">Core {{ i }}</span>
                    <v-progress-linear
                        :model-value="core"
                        :color="utilizationColor(core)"
                        height="3"
                        rounded
                        class="resource-bar"/>
                    <span class="resource-value">{{ core.toFixed(1) }}%</span>
                  </div>
                </div>
              </v-expand-transition>
            </template>

            <template v-if="ruResource.value.memory?.utilization != null">
              <div class="resource-section-label" :class="{'mt-2': ruResource.value.cpu?.utilization != null}">
                Memory
              </div>
              <div class="resource-row">
                <span class="resource-label">Used</span>
                <v-progress-linear
                    :model-value="ruResource.value.memory.utilization"
                    :color="utilizationColor(ruResource.value.memory.utilization)"
                    height="3"
                    rounded
                    class="resource-bar"/>
                <span class="resource-value">{{ ruResource.value.memory.utilization.toFixed(1) }}%</span>
              </div>
            </template>

            <template v-if="ruResource.value.disksList?.length">
              <div class="resource-section-label mt-2 d-flex align-center">
                <span class="flex-grow-1">Disk space</span>
                <button
                    v-if="ruResource.value.disksList.length > 1"
                    class="expand-btn"
                    @click="diskExpanded = !diskExpanded">
                  {{ diskExpanded ? '−' : '+' }}
                </button>
              </div>
              <!-- collapsed: worst disk only -->
              <v-expand-transition>
                <div v-show="!diskExpanded">
                  <div class="resource-row">
                    <v-tooltip :text="worstDisk(ruResource.value.disksList).mountPoint" location="bottom">
                      <template #activator="{ props: _props }">
                        <span class="resource-label text-truncate" v-bind="_props">
                          {{ diskShortLabel(worstDisk(ruResource.value.disksList).mountPoint) }}
                        </span>
                      </template>
                    </v-tooltip>
                    <v-progress-linear
                        v-if="worstDisk(ruResource.value.disksList).utilization != null"
                        :model-value="worstDisk(ruResource.value.disksList).utilization"
                        :color="utilizationColor(worstDisk(ruResource.value.disksList).utilization)"
                        height="3"
                        rounded
                        class="resource-bar"/>
                    <span v-if="worstDisk(ruResource.value.disksList).utilization != null" class="resource-value">
                      {{ worstDisk(ruResource.value.disksList).utilization.toFixed(1) }}%
                    </span>
                  </div>
                </div>
              </v-expand-transition>
              <!-- expanded: all disks -->
              <v-expand-transition>
                <div v-show="diskExpanded">
                  <div
                      v-for="disk in ruResource.value.disksList"
                      :key="disk.mountPoint"
                      class="resource-row">
                    <v-tooltip :text="disk.mountPoint" location="bottom">
                      <template #activator="{ props: _props }">
                        <span class="resource-label text-truncate" v-bind="_props">
                          {{ diskShortLabel(disk.mountPoint) }}
                        </span>
                      </template>
                    </v-tooltip>
                    <v-progress-linear
                        v-if="disk.utilization != null"
                        :model-value="disk.utilization"
                        :color="utilizationColor(disk.utilization)"
                        height="3"
                        rounded
                        class="resource-bar"/>
                    <span v-if="disk.utilization != null" class="resource-value">
                      {{ disk.utilization.toFixed(1) }}%
                    </span>
                  </div>
                </div>
              </v-expand-transition>
            </template>

            <div v-if="ruResource.value.network?.connectionCount != null"
                 class="mt-2 text-caption text-medium-emphasis d-flex align-center">
              <v-icon size="12" class="mr-1">mdi-connection</v-icon>
              {{ ruResource.value.network.connectionCount }} connections
            </div>
          </div>
        </template>
      </with-resource-use>
    </v-card-text>
  </v-card>
</template>

<script setup>
import {getDownloadLogUrl} from '@/api/ui/log.js';
import {triggerDownloadFromUrl} from '@/components/download/download.js';
import {useHasHubSystem, usePullService, usePullServiceMetadata} from '@/composables/services.js';
import {NodeRole} from '@/stores/cohort.js';
import WithResourceUse from '@/traits/resourceUse/WithResourceUse.vue';
import {computed, reactive} from 'vue';
import {useRouter} from 'vue-router';

const props = defineProps({
  node: {
    type: /** @type {typeof CohortNode} */ Object,
    default: () => null
  }
});
const emit = defineEmits(['click:show-certificates', 'click:forget-node']);

const router = useRouter();

const cpuExpanded = defineModel('cpuExpanded', {type: Boolean, default: false});
const diskExpanded = defineModel('diskExpanded', {type: Boolean, default: false});

const {value: logServiceValue} = usePullService(() => ({name: props.node.name + '/systems', id: 'log'}));
const hasLogService = computed(() => !!logServiceValue.value);

const {hasHubSystem, isProxyHub} = useHasHubSystem(() => props.node.name);
const isCentralHub = computed(() => props.node.role === NodeRole.HUB || (hasHubSystem.value && !isProxyHub.value));
const isProxyHubNode = computed(() => hasHubSystem.value && isProxyHub.value && props.node.role !== NodeRole.GATEWAY);

const nodeDetails = reactive({
  automations: usePullServiceMetadata(() => props.node.name + '/automations'),
  drivers: usePullServiceMetadata(() => props.node.name + '/drivers'),
  systems: usePullServiceMetadata(() => props.node.name + '/systems')
});

const accentStyle = computed(() => {
  const colors = [];
  if (props.node.isServer) colors.push('rgb(var(--v-theme-success))');
  if (isCentralHub.value) colors.push('#FB8C00');
  if (props.node.role === NodeRole.GATEWAY) colors.push('rgb(var(--v-theme-secondary))');
  if (isProxyHubNode.value) colors.push('#00ACC1');

  if (colors.length === 0) return {background: 'transparent'};
  if (colors.length === 1) return {background: colors[0]};

  const pct = 100 / colors.length;
  const stops = colors.map((c, i) => {
    const start = (i * pct).toFixed(1);
    const end = ((i + 1) * pct).toFixed(1);
    return i === 0 ? `${c} ${end}%` : `${c} ${start}% ${end}%`;
  });
  return {background: `linear-gradient(to right, ${stops.join(', ')})`};
});

const onShowCertificates = (address) => emit('click:show-certificates', address);
const onForgetNode = (address) => emit('click:forget-node', address);

const onViewLiveLogs = (name) => {
  router.push({path: '/system/logs', query: {node: name}});
};

const onDownloadLogs = async (name) => {
  try {
    const response = await getDownloadLogUrl({name, includeRotated: true});
    for (const file of response.filesList) {
      triggerDownloadFromUrl(file.url, file.filename);
    }
  } catch (e) {
    console.warn('Failed to download logs for', name, e);
  }
};

const diskShortLabel = (mountPoint) => {
  return mountPoint.split('/').filter(Boolean).pop() || mountPoint;
};

const worstDisk = (disks) => {
  return disks.reduce((worst, disk) => {
    if (worst == null) return disk;
    if ((disk.utilization ?? -1) > (worst.utilization ?? -1)) return disk;
    return worst;
  }, null) ?? disks[0];
};

const utilizationColor = (value) => {
  if (value >= 80) return 'error';
  if (value >= 60) return 'warning';
  return 'success';
};
</script>

<style scoped>
.node-card__accent {
  height: 3px;
  width: 100%;
}

.min-width-0 {
  min-width: 0;
}

/* Split chip */
.split-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 12px;
  overflow: hidden;
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
  cursor: default;
}

.split-chip__half {
  padding: 3px 7px;
  color: #fff;
}

.split-chip__half--success {
  background: rgb(var(--v-theme-success));
}

.split-chip__half--orange {
  background: #FB8C00;
}

.split-chip__half--secondary {
  background: rgb(var(--v-theme-secondary));
}

/* Service stats */
.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
}

.stat-box {
  text-align: center;
  background: rgba(var(--v-theme-surface-variant), 0.5);
  border-radius: 6px;
  padding: 6px 4px;
}

.stat-box--error {
  background: rgba(var(--v-theme-error), 0.08);
}

.stat-value {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.2;
}

.stat-label {
  font-size: 10px;
  opacity: 0.55;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-top: 2px;
}

/* Resource usage */
.resource-use {
  font-size: 12px;
}

.resource-section-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: 0.5;
  margin-bottom: 4px;
}

.resource-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.resource-row--core {
  opacity: 0.7;
}

.resource-bar {
  flex: 1;
  max-width: 60px;
}

.resource-label {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  opacity: 0.7;
}

.resource-value {
  width: 40px;
  text-align: right;
  flex-shrink: 0;
  font-size: 11px;
}

.expand-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  padding: 0 2px;
  opacity: 0.5;
  color: inherit;
  font-family: inherit;
}

.expand-btn:hover {
  opacity: 1;
}
</style>
