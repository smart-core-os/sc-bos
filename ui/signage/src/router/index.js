import {useAccountStore} from '@/stores/account.js';
import {useUiConfigStore} from '@/stores/ui-config.js';
import WidgetsDashboard from '@/views/WidgetsDashboard.vue';
import {createRouter, createWebHistory} from 'vue-router';

import AirQuality from '@/views/AirQuality.vue';
import BuildingOccupancy from '@/views/BuildingOccupancy.vue';
import EnergyUsage from '@/views/EnergyUsage.vue';
import TemperatureSystems from '@/views/TemperatureSystems.vue';
import WaterUsage from '@/views/WaterUsage.vue';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/router/login/LoginPage.vue'),
      props: true
    },
    {
      path: '/',
      name: 'dashboard',
      component: WidgetsDashboard,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props', {});
      }
    },
    {
      path: '/water',
      name: 'water',
      component: WaterUsage,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props.water', {});
      },
    },
    {
      path: '/occupancy',
      name: 'occupancy',
      component: BuildingOccupancy,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props.occupancy', {});
      },
    },
    {
      path: '/airquality',
      name: 'airquality',
      component: AirQuality,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props.airQuality', {});
      }
    },
    {
      path: '/energy',
      name: 'energy',
      component: EnergyUsage,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props.energy', {});
      },
    },
    {
      path: '/temperature',
      name: 'temperature',
      component: TemperatureSystems,
      props: () => {
        const uiConfig = useUiConfigStore();
        return uiConfig.getOrDefault('props.temperature', {});
      }
    }
  ]
});

if (window) {
  router.beforeEach(async (to, from, next) => {
      const uiConfig = useUiConfigStore();
      await uiConfig.loadConfig();
      await useAccountStore().initialise(uiConfig.config?.auth?.providers);
      next();
  });
}

export default router;
