<template>
  <div>
    <h3 class="text-h3 pt-2 pb-6">Subsystems</h3>

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
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed} from 'vue';

const uiConfig = useUiConfigStore();
const subsystems = computed(() => uiConfig.getOrDefault('system.subsystemHealth.subsystems', []));
</script>
