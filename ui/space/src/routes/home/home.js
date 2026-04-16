import AirQualityCard from '@/routes/components/AirQualityCard.vue';
import AirTemperatureCard from '@/routes/components/AirTemperatureCard.vue';
import LightCard from '@/routes/components/LightCard.vue';
import {useConfigStore} from '@/stores/config.js';
import {useUiConfigStore} from '@/stores/ui-config.js';
import {computed} from 'vue';

/**
 * @return {{
 *   widgets: import('vue').ComputedRef<{
 *     'is': import('vue').Component,
 *     key: string,
 *     props: Record<string, any>
 *   }[]>
 * }}
 */
export function useHomeConfig() {
  const uiConfigStore = useUiConfigStore();
  const uiConfig = computed(() => uiConfigStore.config);
  const homePageConfig = computed(() => uiConfig.value?.pages?.home);
  const widgetConfig = computed(() => homePageConfig.value?.widgets ?? []);

  const appConfigStore = useConfigStore();
  const zoneId = computed(() => appConfigStore.activeZoneId);

  const availableWidgets = {
    'air-quality': {
      is: AirQualityCard,
      props() {
        return {name: zoneId.value};
      }
    },
    'lighting': {
      is: LightCard,
      props() {
        return {name: zoneId.value};
      }
    },
    'temperature': {
      is: AirTemperatureCard,
      props() {
        return {name: zoneId.value};
      }
    }
  };
  const widgets = computed(() => {
    return widgetConfig.value
        .map(cfg => {
          const key = cfg.name;
          const w = availableWidgets[key];
          if (!w) {
            console.warn(`Unknown widget: ${key}`);
            return null;
          }
          return {
            is: w.is,
            key,
            props: w.props()
          };
        })
        .filter(Boolean);
  });

  return {
    widgets
  }
}
