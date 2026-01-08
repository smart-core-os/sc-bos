import {defineAsyncComponent} from 'vue';

export const builtinLayouts = {
  'LayoutMainSide': defineAsyncComponent(() => import('@/dynamic/layout/LayoutMainSide.vue')),
  'LayoutGrid': defineAsyncComponent(() => import('@/dynamic/layout/LayoutGrid.vue')),
  // full pages, not widget containers
  'page/AirQuality': defineAsyncComponent(() => import('@/routes/ops/air-quality/AirQuality.vue')),
  'page/DynamicDownloads': defineAsyncComponent(() => import('@/routes/ops/downloads/DownloadFileCard.vue')),
  'page/EmergencyLighting': defineAsyncComponent(() => import('@/routes/ops/emergency-lighting/EmergencyLighting.vue')),
  'page/EmergencyLightingV2': defineAsyncComponent(() => import('@/routes/ops/emergency-lighting/EmergencyLightingV2.vue')),
  'page/Reports': defineAsyncComponent(() => import('@/routes/ops/reports/Reports.vue')),
  'page/Security': defineAsyncComponent(() => import('@/routes/ops/security/SecurityHome.vue')),
  'page/SecurityEvents': defineAsyncComponent(() => import('@/routes/ops/security-events/SecurityEventsTable.vue')),
  'page/Waste': defineAsyncComponent(() => import('@/routes/ops/waste/WasteTable.vue'))
};
