<template>
  <v-card width="300px" class="ma-2">
    <v-card-title
        class="text-body-large font-weight-bold d-flex align-center text-wrap"
        style="word-break: break-all">
      {{ node.name }}
      <v-spacer/>
      <v-menu min-width="175px">
        <template #activator="{ props: _props }">
          <v-btn
              icon="mdi-dots-vertical"
              variant="text"
              size="small"
              v-bind="_props">
            <v-icon size="24"/>
          </v-btn>
        </template>
        <v-list class="py-0">
          <v-list-item link @click="onShowCertificates(node.grpcAddress)">
            <v-list-item-title>
              View Certificate
            </v-list-item-title>
          </v-list-item>
          <v-list-item v-if="node.role !== NodeRole.HUB && !node.isServer"
                       link
                       @click="onForgetNode(node.grpcAddress)">
            <v-list-item-title class="text-error">
              Forget Node
            </v-list-item-title>
          </v-list-item>
          <v-list-item v-if="canReboot" link @click="showRebootDialog = true">
            <v-list-item-title class="text-warning">Reboot Node</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-card-title>
    <v-card-subtitle v-if="node.description !== ''">{{ node.description }}</v-card-subtitle>

    <v-card-text>
      <v-list density="compact">
        <v-list-item
            class="pa-0"
            style="min-height: 20px">
          {{ node.grpcAddress }}
        </v-list-item>
        <v-list-item
            v-for="(response, service) in nodeDetails"
            :key="service"
            :class="[{'text-red': response.streamError}, 'pa-0 ma-0']"
            style="min-height: 20px">
          <span class="mr-1 text-capitalize">{{ service }}: {{ response.value?.totalActiveCount }}</span>
          <status-alert :resource="response.streamError"/>
        </v-list-item>
        <v-list-item v-if="rebootState" class="pa-0" style="min-height: 20px">
          <span>Uptime: {{ uptime ?? '—' }}</span>
        </v-list-item>
        <v-list-item v-if="rebootState?.lastRebootReason" class="pa-0" style="min-height: 20px">
          <span class="text-truncate">Last Reboot Reason: {{ rebootState.lastRebootReason }}</span>
        </v-list-item>
        <v-list-item v-if="rebootState?.lastRebootActor?.displayName" class="pa-0" style="min-height: 20px">
          <span>Last actor: {{ rebootState.lastRebootActor.displayName }}</span>
        </v-list-item>
      </v-list>
      <with-resource-use :name="node.name" :paused="false" v-slot="{ resource: ruResource }">
        <template v-if="!ruResource.streamError && ruResource.value">
          <v-list-item class="pa-0" style="min-height: 20px">
            <span v-if="ruResource.value.cpu?.utilization != null">
              CPU: {{ ruResource.value.cpu.utilization.toFixed(1) }}%
            </span>
            <span v-if="ruResource.value.memory?.utilization != null" class="ml-2">
              Mem: {{ ruResource.value.memory.utilization.toFixed(1) }}%
            </span>
          </v-list-item>
          <v-list-item v-if="ruResource.value.network?.connectionCount != null" class="pa-0" style="min-height: 20px">
            <span>SC-BOS connections: {{ ruResource.value.network.connectionCount }}</span>
          </v-list-item>
          <template v-if="ruResource.value.disksList?.length">
            <v-list-item class="pa-0 mt-1" style="min-height: 20px">
              <span class="text-caption text-medium-emphasis">Disks</span>
            </v-list-item>
            <v-list-item
                v-for="disk in ruResource.value.disksList"
                :key="disk.mountPoint"
                class="pa-0"
                style="min-height: 20px">
              <span>{{ disk.mountPoint }}</span>
              <span v-if="disk.utilization != null" class="ml-2">
                {{ disk.utilization.toFixed(1) }}% used
              </span>
            </v-list-item>
          </template>
        </template>
      </with-resource-use>
      <div class="chips">
        <v-chip
            v-if="node.isServer"
            color="success"
            size="small"
            variant="flat"
            v-tooltip:bottom="'The component you are connected to'">
          connected
        </v-chip>
        <v-chip v-if="node.role === NodeRole.GATEWAY" color="accent" size="small" variant="flat">gateway</v-chip>
        <v-chip v-if="node.role === NodeRole.HUB" color="primary" size="small" variant="flat">hub</v-chip>
      </div>
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
import StatusAlert from '@/components/StatusAlert.vue';
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

const {hasAnyRole} = useAuthSetup();
const canReboot = computed(() => hasAnyRole('admin', 'superAdmin'));
const accountStore = useAccountStore();

const nodeDetails = reactive({
  automations: usePullServiceMetadata(() => props.node.name + '/automations'),
  drivers: usePullServiceMetadata(() => props.node.name + '/drivers'),
  systems: usePullServiceMetadata(() => props.node.name + '/systems')
});

// Reboot state stream
const rebootResource = reactive(newResourceValue());
onScopeDispose(() => closeResource(rebootResource));
watchResource(
    () => ({name: props.node.name}),
    () => false,
    (req) => {
      pullBootState(req, rebootResource);
      return () => closeResource(rebootResource);
    }
);

const rebootState = computed(() => rebootResource.value);

// Live uptime counter
const now = ref(Date.now());
const uptimeInterval = setInterval(() => {
  now.value = Date.now();
}, 1000);
onScopeDispose(() => clearInterval(uptimeInterval));

const uptime = computed(() => {
  const bt = rebootState.value?.bootTime;
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

const onShowCertificates = (address) => {
  emit('click:show-certificates', address);
};
const onForgetNode = (address) => {
  emit('click:forget-node', address);
};

</script>

<style scoped>
.chips > :not(:last-child) {
  margin-right: 4px;
}
</style>
