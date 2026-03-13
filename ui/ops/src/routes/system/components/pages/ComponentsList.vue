<template>
  <div>
    <v-row class="ml-0 pl-0 my-0">
      <h3 class="text-h3 pt-2 pb-6">Components</h3>
      <v-spacer/>
      <v-tooltip location="left">
        <template #activator="{ props }">
          <v-btn
              class="mr-4"
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
    </v-row>

    <div class="d-flex flex-wrap ml-n2">
      <keycloak-card v-if="keycloakHealth.isConfigured"/>
      <cohort-node-card
          v-for="node in cohortNodes"
          :key="node.address"
          :node="node"
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
import {useCohortStore} from '@/stores/cohort.js';
import {useKeycloakHealthStore} from '@/stores/keycloakHealth.js';
import {storeToRefs} from 'pinia';
import {ref, watch} from 'vue';

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
