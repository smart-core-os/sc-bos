import {builtinLayouts} from '@/dynamic/layout/pallet.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {findActiveItem} from '@/util/router.js';
import {builtinWidgets} from '@/dynamic/widgets/pallet.js';
import {computed, markRaw, toValue} from 'vue';

/**
 * Returns a modern page config object based on the legacy `ops.overview.children` json.
 *
 * @param {MaybeRefOrGetter<string|string[]>} path
 * @return {{
 *   layout: import('vue').Component,
 *   main: ComputedRef<{component:*,props:Object}[]>,
 *   after: ComputedRef<{component:*,props:Object}[]>,
 *   title: ComputedRef<string>
 * }}
 */
export default function useDashPage(path) {
  const uiConfig = useUiConfigStore();

  // uri encoded path segments from path.
  const pathSegments = computed(() => {
    let p = toValue(path);
    if (!Array.isArray(p)) p = p.split('/');
    return p.map(s => encodeURIComponent(s));
  });

  const pageConfigRoot = computed(
      () => uiConfig.getOrDefault('ops.overview', {}));
  // the current page config, detailing the title, icon and widgets to show.
  const pageConfig = computed(() => {
    if (!pathSegments.value.length || !pageConfigRoot.value?.children?.length) return null;
    // Note: findActiveItem takes uri-encoded path segments
    return findActiveItem(pageConfigRoot.value.children, pathSegments.value);
  });

  // config for all the widgets we support
  const powerHistoryConfig = computed(() => {
    const v = pageConfig.value?.widgets?.showEnergyConsumption;
    if (!v) return false;
    return {
      demand: /** @type {string} */ v,
      hideTotal: true,
      chartTitle: 'Energy Consumption'
    };
  });
  const zoneNotificationConfig = computed(() => {
    const v = pageConfig.value?.widgets?.showNotifications;
    if (!v) return false;
    return {
      forceQuery: {zone: /** @type {string} */ v}
    };
  });
  const presenceConfig = computed(() => {
    const v = pageConfig.value?.widgets?.showOccupancy;
    if (!v) return false;
    return {
      source: /** @type {string} */ v
    };
  });
  const environmentConfig = computed(() => {
    const v = pageConfig.value?.widgets?.showEnvironment;
    if (!v) return false;
    return {
      internal: /** @type {string} */ v.indoor,
      external: /** @type {string} */ v.outdoor,
      sound: /** @type {string} */ v.sound
    };
  });

  const title = computed(() => pageConfig.value?.title ?? '');
  const extendedTitle = computed(() => {
    if (!title.value) return '';
    return title.value + ' Status Overview';
  });

  const addIfPresent = (arr, props, component) => {
    props = toValue(props);
    if (props) {
      arr.push({props, component: markRaw(component)});
    }
  };
  return {
    layout: markRaw(builtinLayouts['LayoutMainSide']),
    title: extendedTitle,
    main: computed(() => {
      const res = [];
      addIfPresent(res, powerHistoryConfig, builtinWidgets['power-history/PowerHistoryCard']);
      addIfPresent(res, zoneNotificationConfig, builtinWidgets['notifications/ZoneNotifications']);
      return res;
    }),
    after: computed(() => {
      const res = [];
      addIfPresent(res, presenceConfig, builtinWidgets['occupancy/PresenceCard']);
      addIfPresent(res, environmentConfig, builtinWidgets['environmental/EnvironmentalCard']);
      return res;
    })
  };
}
