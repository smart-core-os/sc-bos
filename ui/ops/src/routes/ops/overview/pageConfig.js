import {computed, defineAsyncComponent, markRaw, reactive, toValue} from 'vue';
import {builtinLayouts} from '@/dynamic/layout/pallet.js';
const PlaceholderCard = defineAsyncComponent(() => import('@/dynamic/widgets/general/PlaceholderCard.vue'));
import {builtinWidgets} from '@/dynamic/widgets/pallet.js';
import useBuildingConfig from '@/routes/ops/overview/pages/buildingConfig.js';
import useDashPage from '@/routes/ops/overview/pages/dashPage.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {findActiveItem} from '@/util/router.js';

/**
 * @param {MaybeRefOrGetter<string|string[]>} path - uri decoded path to the page
 * @return {{
 *   layout: ComputedRef<import('vue').Component>,
 *   config: ComputedRef<Object>,
 *   isLegacySubPage: ComputedRef<boolean>,
 *   isLegacyOverview: ComputedRef<boolean>,
 *   pageConfig: ComputedRef<undefined|*|OverviewChild[]|null>,
 *   pageConfigNorm: ComputedRef<{}>
 * }}
 */
export default function usePageConfig(path) {
  const uiConfig = useUiConfigStore();

  // uri encoded path segments from path.
  const pathSegments = computed(() => {
    let p = toValue(path);
    if (!Array.isArray(p)) p = p.split('/');
    return p.map(s => encodeURIComponent(s));
  });

  // The config object (in ops.paths) described by path.
  // If ops.pages is not defined - i.e. legacy config is being used - this will also be undefined.
  const pageConfig = computed(() => {
    const pages = uiConfig.getOrDefault('ops.pages');
    if (!pages) {
      return undefined; // fall back to legacy config
    }
    if (!pathSegments.value.length) {
      return pages[0];
    }
    return findActiveItem(pages, pathSegments.value);
  });
  // The same as pageConfig but with special values hydrated to their component references.
  const pageConfigNorm = computed(() => {
    const cfg = pageConfig.value;
    if (!cfg) return cfg;

    const norm = (o, k = null) => {
      if (Array.isArray(o)) {
        return o.map(v => norm(v));
      } else if (typeof o === 'object') {
        return Object.entries(o).reduce((acc, [k, v]) => {
          acc[k] = norm(v, k);
          return acc;
        }, {});
      } else if (typeof o === 'string') {
        if (o.startsWith('builtin:')) {
          const [, builtin] = o.split(':');
          switch (k) {
            case 'layout': {
              const comp = builtinLayouts[builtin];
              if (!comp) {
                console.warn(`Unknown layout: ${builtin}`);
                return PlaceholderCard;
              }
              return markRaw(comp);
            }
            case 'component': {
              const comp = builtinWidgets[builtin];
              if (!comp) {
                console.warn(`Unknown widget: ${builtin}`);
                return PlaceholderCard;
              }
              return markRaw(comp);
            }
          }
        }
      }
      return o;
    };

    return norm(cfg);
  });

  const isLegacyOverview = computed(() => !pageConfig.value && pathSegments.value.length === 1);
  const isLegacySubPage = computed(() => !pageConfig.value && pathSegments.value.length > 1);

  // Page config taken from either the newer ops.pages or legacy ops.overview as needed.
  const configObj = computed(() => {
    if (isLegacyOverview.value) {
      return reactive(useBuildingConfig());
    }
    if (isLegacySubPage.value) {
      return reactive(useDashPage(() => pathSegments.value
          // skip /building prefix in segments as useDashPage isn't expecting it
          .slice(1)
          // decode segments just like vue-router does
          .map(s => decodeURIComponent(s))));
    }
    return pageConfigNorm.value ?? {};
  });

  // We filter out special properties of the config when binding it to elements.
  // Without this we'd be trying to bind props like `children` which is a DOM property.
  const filterProps = {'layout': true, 'children': true};
  const filteredConfig = computed(() => {
    const config = configObj.value;
    if (!config) return {};
    return Object.entries(config)
        .reduce((acc, [k, v]) => {
          if (Object.hasOwn(filterProps, k)) return acc;
          acc[k] = v;
          return acc;
        }, {});
  });

  return {
    pageConfig,
    pageConfigNorm,
    isLegacyOverview,
    isLegacySubPage,
    layout: computed(() => configObj.value?.layout),
    config: filteredConfig
  };
}
