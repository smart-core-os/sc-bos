<template>
  <div style="position: relative;">
    <div class="d-flex align-start mb-2">
      <h3 class="text-h3 pt-2 pb-6">Components</h3>
      <v-spacer/>
      <div class="d-flex align-start" style="gap: 8px">
        <v-tooltip location="bottom">
          <template #activator="{ props }">
            <v-btn
                class="mt-2"
                color="primary"
                icon="mdi-plus"
                size="small"
                v-bind="props"
                @click="showModal = true">
              <v-icon size="24"/>
            </v-btn>
          </template>
          Enroll Node
        </v-tooltip>
        <keycloak-card v-if="keycloakHealth.isConfigured" class="flex-shrink-0"/>
        <database-card class="flex-shrink-0"/>
      </div>
    </div>

    <div class="d-flex flex-wrap ml-n2">
      <cohort-node-card
          v-for="node in cohortNodes"
          :key="node.address"
          :node="node"
          v-model:cpu-expanded="cpuExpanded"
          v-model:disk-expanded="diskExpanded"
          @click:show-certificates="onShowCertificates"
          @click:forget-node="onForgetNode"/>
    </div>

    <!-- Modal -->
    <enroll-hub-node-modal
        v-model:show-modal="showModal"
        v-model:node-query="nodeQuery"
        :list-items="cohortNodes"/>
  </div>
</template>

<script setup>
import CohortNodeCard from '@/routes/system/components/CohortNodeCard.vue';
import EnrollHubNodeModal from '@/routes/system/components/EnrollHubNodeModal.vue';
import KeycloakCard from '@/routes/system/components/KeycloakCard.vue';
import DatabaseCard from '@/routes/system/components/DatabaseCard.vue';
import {useCohortStore} from '@/stores/cohort.js';
import {useKeycloakHealthStore} from '@/stores/keycloakHealth.js';
import {storeToRefs} from 'pinia';
import {ref, watch} from 'vue';

const cpuExpanded = ref(false);
const diskExpanded = ref(false);

const showModal = ref(false);

const {cohortNodes} = storeToRefs(useCohortStore());
const keycloakHealth = useKeycloakHealthStore();

const nodeQuery = ref({
  address: null,
  isQueried: false,
  isToForget: false
});

const onShowCertificates = (address) => {
  nodeQuery.value.address = address;
  nodeQuery.value.isQueried = true;
  nodeQuery.value.isToForget = false;
  showModal.value = true;
};

const onForgetNode = (address) => {
  nodeQuery.value.address = address;
  nodeQuery.value.isQueried = false;
  nodeQuery.value.isToForget = true;
  showModal.value = true;
};

watch(showModal, (newModal) => {
  if (newModal === false) {
    nodeQuery.value.address = null;
    nodeQuery.value.isQueried = false;
    nodeQuery.value.isToForget = false;
  }
}, {immediate: true, deep: true, flush: 'sync'});
</script>

<style scoped>

</style>
