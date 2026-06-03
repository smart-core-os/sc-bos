<template>
  <div>
    <div class="d-flex align-center mb-0">
      <h3 class="text-h3 pt-2 pb-6 flex-grow-1">Subsystems</h3>
      <v-tooltip v-if="subsystems.length > 0" location="bottom">
        <template #activator="{ props }">
          <v-btn
              class="mt-n3"
              icon="mdi-download"
              size="small"
              :loading="exporting"
              v-bind="props"
              @click="onExportReport">
            <v-icon size="24"/>
          </v-btn>
        </template>
        Export Report
      </v-tooltip>
    </div>

    <div v-if="subsystems.length > 0" class="d-flex flex-wrap ml-n2">
      <subsystem-health-card
          v-for="subsystem in subsystems"
          :key="subsystem.title"
          :title="subsystem.title"
          :icon="subsystem.icon"
          :description="subsystem.description"
          :checks="subsystem.checks"
          class="ma-2"/>
    </div>
    <div v-else class="text-body-2 text-medium-emphasis">
      No subsystems configured. Add entries to
      <code>config.system.subsystemHealth.subsystems</code> in your UI configuration.
    </div>
  </div>
</template>

<script setup>
import SubsystemHealthCard from '@/routes/system/components/SubsystemHealthCard.vue';
import {downloadSubsystemHealthReport} from './subsystemHealthExport.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed, ref} from 'vue';

const uiConfig = useUiConfigStore();
const subsystems = computed(() => uiConfig.getOrDefault('system.subsystemHealth.subsystems', []));

const exporting = ref(false);
const onExportReport = async () => {
  exporting.value = true;
  try {
    await downloadSubsystemHealthReport(subsystems.value);
  } finally {
    exporting.value = false;
  }
};
</script>
