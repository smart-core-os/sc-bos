<template>
  <v-card width="200px" class="ma-2 keycloak-card">
    <div class="keycloak-card__accent"/>
    <v-card-text class="pa-2 pb-2" style="position: relative; overflow: hidden;">

      <!-- Shield watermark -->
      <v-icon class="keycloak-card__watermark">mdi-shield-account</v-icon>

      <!-- Header -->
      <div class="d-flex align-start" style="position: relative;">
        <div class="flex-grow-1 min-width-0">
          <div class="d-flex align-center text-subtitle-2 font-weight-bold" style="gap: 6px">
            <span class="text-truncate" :title="keycloakHealth.realm">{{ keycloakHealth.realm }}</span>
            <v-chip size="x-small" variant="flat" color="#1565C0" class="flex-shrink-0">sso</v-chip>
          </div>
          <div class="text-caption text-medium-emphasis text-truncate mt-1" :title="keycloakHealth.url">
            {{ keycloakHealth.url }}
          </div>
        </div>
        <v-btn
            icon="mdi-refresh"
            variant="text"
            size="x-small"
            density="compact"
            :loading="keycloakHealth.pending"
            v-tooltip:bottom="'Check now'"
            @click="keycloakHealth.pollNow()">
          <v-icon size="14"/>
        </v-btn>
      </div>

      <v-divider class="mt-1 mb-1"/>

      <!-- Status -->
      <div class="status-row" style="position: relative;">
        <template v-if="keycloakHealth.pending">
          <v-progress-circular size="12" width="2" indeterminate color="#1565C0" class="mr-2"/>
          <span class="text-caption">Checking…</span>
        </template>
        <template v-else-if="keycloakHealth.error">
          <v-icon size="14" color="error" class="mr-1">mdi-alert-circle</v-icon>
          <span class="text-caption text-error">Unreachable</span>
        </template>
        <template v-else>
          <v-icon size="14" color="success" class="mr-1">mdi-check-circle</v-icon>
          <span class="text-caption text-success">Reachable</span>
        </template>
      </div>

      <div v-if="keycloakHealth.lastPoll" class="text-caption text-medium-emphasis" style="position: relative;">
        {{ lastCheckedLabel }}
      </div>

    </v-card-text>
  </v-card>
</template>

<script setup>
import {useKeycloakHealthStore} from '@/stores/keycloakHealth.js';
import {computed, onUnmounted, ref} from 'vue';

const keycloakHealth = useKeycloakHealthStore();

// Reactive "X seconds ago" label
const now = ref(Date.now());
const timer = setInterval(() => { now.value = Date.now(); }, 10_000);
onUnmounted(() => clearInterval(timer));

const lastCheckedLabel = computed(() => {
  if (!keycloakHealth.lastPoll) return '';
  const secs = Math.round((now.value - keycloakHealth.lastPoll) / 1000);
  if (secs < 10) return 'Checked just now';
  if (secs < 60) return `Checked ${secs}s ago`;
  return `Checked ${Math.round(secs / 60)}m ago`;
});
</script>

<style scoped>
.keycloak-card__accent {
  height: 3px;
  width: 100%;
  background: #1565C0;
}

.keycloak-card__watermark {
  position: absolute;
  bottom: -12px;
  right: -8px;
  font-size: 96px !important;
  opacity: 0.04;
  color: #1565C0 !important;
  pointer-events: none;
  user-select: none;
}

.min-width-0 {
  min-width: 0;
}

.status-row {
  display: flex;
  align-items: center;
}
</style>