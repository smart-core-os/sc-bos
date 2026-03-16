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
            <!-- Split chip: connected + hub -->
            <span v-if="node.isServer && node.role === NodeRole.HUB"
                  class="split-chip"
                  v-tooltip:bottom="'Connected hub'">
              <span class="split-chip__half split-chip__half--success">connected</span>
              <span class="split-chip__half split-chip__half--primary">hub</span>
            </span>
            <!-- Split chip: connected + gateway -->
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
              <v-chip v-if="node.role === NodeRole.HUB" color="primary" size="x-small" variant="flat">
                hub
              </v-chip>
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
            <v-list-item v-if="node.role !== NodeRole.HUB && !node.isServer"
                         link
                         @click="onForgetNode(node.grpcAddress)">
              <v-list-item-title class="text-error">Forget Node</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="canReboot" link @click="showRebootDialog = true">
              <v-list-item-title class="text-warning">Reboot Node</v-list-item-title>
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

      <!-- Boot info -->
      <template v-if="bootState">
        <v-divider class="mt-2 mb-3"/>
        <div class="resource-use">
          <div class="resource-section-label">Last boot</div>
          <div v-if="uptime" class="resource-row">
            <span class="resource-label">Uptime</span>
            <span class="resource-value" style="width: auto">{{ uptime }}</span>
          </div>
          <div v-if="bootState.lastRebootReason" class="resource-row">
            <span class="resource-label">Reason</span>
            <span class="resource-value text-truncate" style="width: auto" :title="bootState.lastRebootReason">
              {{ bootState.lastRebootReason }}
            </span>
          </div>
          <div v-if="bootState.lastRebootActor?.displayName" class="resource-row">
            <span class="resource-label">Actor</span>
            <span class="resource-value text-truncate" style="width: auto"
                  :title="bootState.lastRebootActor.displayName">
              {{ bootState.lastRebootActor.displayName }}
            </span>
          </div>
        </div>
      </template>

    </v-card-text>
  </v-card>

  <v-dialog v-model="showRebootDialog" max-width="400px">
    <v-card>
      <v-card-title>Reboot {{ node.name }}?</v-card-title>
      <v-card-text>
        <p class="mb-2 text-body-2">
          The node will exit and be restarted by its supervisor.
          You may lose connection briefly.
        </p>
        <v-text-field v-model="rebootReason" label="Reason (optional)" density="compact"/>
        <v-alert v-if="rebootTracker.error" type="error" density="compact" class="mt-2">
          {{ rebootTracker.error }}
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer/>
        <v-btn @click="showRebootDialog = false">Cancel</v-btn>
        <v-btn color="warning" :loading="rebootTracker.loading" @click="confirmReboot">
          Reboot
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import {pullBootState, reboot} from '@/api/sc/traits/boot.js';
import {closeResource, newResourceValue} from '@/api/resource.js';
import useAuthSetup from '@/composables/useAuthSetup.js';
import {usePullServiceMetadata} from '@/composables/services.js';
import {useAccountStore} from '@/stores/account.js';
import {NodeRole} from '@/stores/cohort.js';
import WithResourceUse from '@/traits/resourceUse/WithResourceUse.vue';
import {watchResource} from '@/util/traits.js';
import {computed, onScopeDispose, reactive, ref} from 'vue';

const props = defineProps({
  node: {
    type: /** @type {typeof CohortNode} */ Object,
    default: () => null
  }
});
const emit = defineEmits(['click:show-certificates', 'click:forget-node']);

const cpuExpanded = defineModel('cpuExpanded', {type: Boolean, default: false});
const diskExpanded = defineModel('diskExpanded', {type: Boolean, default: false});

const {hasAnyRole} = useAuthSetup();
const canReboot = computed(() => hasAnyRole('admin', 'superAdmin'));
const accountStore = useAccountStore();

const nodeDetails = reactive({
  automations: usePullServiceMetadata(() => props.node.name + '/automations'),
  drivers: usePullServiceMetadata(() => props.node.name + '/drivers'),
  systems: usePullServiceMetadata(() => props.node.name + '/systems')
});

const accentStyle = computed(() => {
  const colors = [];
  if (props.node.isServer) colors.push('rgb(var(--v-theme-success))');
  if (props.node.role === NodeRole.HUB) colors.push('rgb(var(--v-theme-primary))');
  if (props.node.role === NodeRole.GATEWAY) colors.push('rgb(var(--v-theme-secondary))');

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

// Boot state stream
const bootResource = reactive(newResourceValue());
onScopeDispose(() => closeResource(bootResource));
watchResource(
    () => ({name: props.node.name}),
    () => false,
    (req) => {
      pullBootState(req, bootResource);
      return () => closeResource(bootResource);
    }
);
const bootState = computed(() => bootResource.value);

const now = ref(Date.now());
const uptimeInterval = setInterval(() => { now.value = Date.now(); }, 1000);
onScopeDispose(() => clearInterval(uptimeInterval));

const uptime = computed(() => {
  const bt = bootState.value?.bootTime;
  if (!bt) return null;
  const bootMs = (bt.seconds * 1000) + Math.floor(bt.nanos / 1e6);
  const diff = Math.floor((now.value - bootMs) / 1000);
  const d = Math.floor(diff / 86400);
  const h = Math.floor((diff % 86400) / 3600);
  const m = Math.floor((diff % 3600) / 60);
  const s = diff % 60;
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
});

// Reboot dialog
const showRebootDialog = ref(false);
const rebootReason = ref('');
const rebootTracker = reactive({loading: false, response: null, error: null});

/** Sends the reboot request and closes the dialog on success. */
function confirmReboot() {
  const actor = (accountStore.email || accountStore.fullName)
      ? {displayName: accountStore.fullName, email: accountStore.email}
      : undefined;
  reboot({name: props.node.name, reason: rebootReason.value, actor}, rebootTracker)
      .then(() => {
        showRebootDialog.value = false;
        rebootReason.value = '';
      })
      .catch(() => { /* error shown in dialog via rebootTracker.error */ });
}
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

.split-chip__half--primary {
  background: rgb(var(--v-theme-primary));
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
