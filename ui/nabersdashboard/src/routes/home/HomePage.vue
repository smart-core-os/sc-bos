<template>
  <img src="@/assets/powered-by-sc.svg" alt="Powered by Smart Core" class="powered-by-logo">
  <div class="home">
    <div class="dashboard-wrapper">
      <SensorDashboard v-if="ready && sensors.length" :sensors="sensors" style="height: 100%;"/>
      <div v-else-if="ready && !sensors.length" class="d-flex justify-center align-center" style="height: 100%;">
        <v-alert type="warning" text="No sensors configured. Update dashboards-config.json to add sensors."/>
      </div>
      <div v-else class="d-flex justify-center align-center" style="height: 100%;">
        <v-progress-circular indeterminate size="48"/>
      </div>
    </div>
  </div>
</template>

<script setup>
import SensorDashboard from '@/components/SensorDashboard.vue';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {useAuthStore} from '@/stores/auth.js';
import {onMounted, computed, ref} from 'vue';

const uiConfig = useUiConfigStore();

const config  = computed(() => uiConfig.config);
const sensors = computed(() => config.value.sensors || []);
const ready   = ref(false);

onMounted(async () => {
  await uiConfig.loadConfig();
  await useAuthStore().fetchToken();
  ready.value = true;
});
</script>

<style scoped>
.home {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-bottom: 0;
}

.dashboard-wrapper {
  flex: 1;
  width: 100%;
  max-width: none;
  margin: 0;
  padding: 0;
  min-height: 0;
}

.powered-by-logo {
  position: fixed;
  bottom: 24px;
  right: 40px;
  width: 320px;
  opacity: 0.45;
  pointer-events: none;
}
</style>
