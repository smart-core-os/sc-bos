<template>
  <v-card width="300px" class="ma-2">
    <v-card-title
        class="text-body-large font-weight-bold d-flex align-center text-wrap"
        style="word-break: break-all">
      {{ node.name }}
      <v-spacer/>
      <template v-if="node.role !== NodeRole.INDEPENDENT">
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
          </v-list>
        </v-menu>
      </template>
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
</template>

<script setup>
import StatusAlert from '@/components/StatusAlert.vue';
import {usePullServiceMetadata} from '@/composables/services.js';
import {NodeRole} from '@/stores/cohort.js';
import WithResourceUse from '@/traits/resourceUse/WithResourceUse.vue';
import {reactive} from 'vue';

const props = defineProps({
  node: {
    type: /** @type {typeof CohortNode} */ Object,
    default: () => null
  }
});
const emit = defineEmits(['click:show-certificates', 'click:forget-node']);

const nodeDetails = reactive({
  automations: usePullServiceMetadata(() => props.node.name + '/automations'),
  drivers: usePullServiceMetadata(() => props.node.name + '/drivers'),
  systems: usePullServiceMetadata(() => props.node.name + '/systems')
});

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
