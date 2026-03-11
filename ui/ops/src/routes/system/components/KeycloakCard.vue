<template>
  <v-card width="300px" class="ma-2">
    <v-card-title
        class="text-body-large font-weight-bold d-flex align-center text-wrap"
        style="word-break: break-all">
      {{ keycloakHealth.realm }}
    </v-card-title>
    <v-card-subtitle>{{ keycloakHealth.url }}</v-card-subtitle>

    <v-card-text>
      <v-list density="compact">
        <v-list-item
            class="pa-0"
            style="min-height: 20px">
          <v-progress-circular v-if="keycloakHealth.pending" size="22" indeterminate/>
          <status-alert
              v-else
              v-bind="statusAttrs"/>
          <span class="ml-1">SSO</span>
        </v-list-item>
      </v-list>
      <div class="chips">
        <v-chip color="primary" size="small" variant="flat">auth</v-chip>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup>
import StatusAlert from '@/components/StatusAlert.vue';
import {useKeycloakHealthStore} from '@/stores/keycloakHealth.js';
import {computed} from 'vue';

const keycloakHealth = useKeycloakHealthStore();

const statusAttrs = computed(() => {
  if (keycloakHealth.error) {
    return {
      color: 'error',
      icon: 'mdi-close',
      resource: {error: keycloakHealth.error}
    };
  }
  return {
    color: 'success',
    icon: 'mdi-check',
    resource: {error: {code: 0, message: `SSO is reachable at ${keycloakHealth.url}`}}
  };
});
</script>

<style scoped>
.chips > :not(:last-child) {
  margin-right: 4px;
}
</style>
