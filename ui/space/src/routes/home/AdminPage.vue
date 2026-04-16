<template>
  <div class="admin-page fill-height d-flex flex-column">
    <!-- Zone list view -->
    <template v-if="!activeZone">
      <div class="pa-8">
        <glance-widget @admin-click="handle10Click"/>
        <notification-toast :show-alert="alertMessage.show" :message="alertMessage.message"/>
      </div>
      <div class="pa-8 pt-0">
        <list-selector title="Select a room" @zone-selected="onZoneSelected"/>
      </div>
    </template>

    <!-- Controls view -->
    <template v-else>
      <div class="pa-8 d-flex flex-column fill-height" style="gap: 1.5em">
        <div class="d-flex align-center">
          <v-btn icon variant="text" @click="activeZone = null">
            <v-icon>mdi-arrow-left</v-icon>
          </v-btn>
          <span class="text-h5 ml-2">{{ activeZoneName }}</span>
        </div>
        <v-spacer/>
        <component
            v-for="widget in widgets"
            :key="widget.key"
            :is="widget.is"
            v-bind="widget.props"/>
        <v-spacer/>
        <img :src="uiConfigStore.theme.logoUrl" class="logo pt-3" alt="Smart Core logo">
      </div>
    </template>
  </div>
</template>

<script setup>
import NotificationToast from '@/components/NotificationToast.vue';
import GlanceWidget from '@/routes/components/GlanceWidget.vue';
import ListSelector from '@/routes/setup/components/ListSelector.vue';
import {useConfigStore} from '@/stores/config';
import {useUiConfigStore} from '@/stores/ui-config.js';
import {computed, ref} from 'vue';
import {useHomeConfig} from './home';

const uiConfigStore = useUiConfigStore();
const configStore = useConfigStore();

const activeZone = ref(null);
const activeZoneName = computed(() =>
  activeZone.value?.metadata?.appearance?.title ?? activeZone.value?.id ?? ''
);

/**
 * @param {{id: string, metadata: object}} zone
 */
function onZoneSelected(zone) {
  activeZone.value = zone;
}

// Widgets driven by ui-config, same as HomePage, but using the temporarily selected zone
const {widgets} = useHomeConfig(computed(() => activeZone.value?.id ?? ''));

// 10-click reconfigure trigger — same pattern as HomePage
const clickCount = ref(0);
const alertMessage = computed(() => {
  if (clickCount.value >= 5 && clickCount.value < 10) {
    return {show: true, message: `${10 - clickCount.value} clicks left for admin menu.`};
  }
  return {show: false, message: ''};
});
let clickTimeout;

const handle10Click = () => {
  clearTimeout(clickTimeout);
  clickCount.value += 1;
  if (clickCount.value === 10) {
    configStore.reconfigure();
    clickCount.value = 0;
  }
  clickTimeout = setTimeout(() => {
    clickCount.value = 0;
  }, 1000);
};
</script>

<style scoped>
.logo {
  width: 100%;
  max-height: 40px;
}
</style>
