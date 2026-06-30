import PluginPageLoading from '@/dynamic/PluginPageLoading.vue';
import {useSidebarStore} from '@/stores/sidebar.js';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {serviceName} from '@/util/gateway.js';
import deepEqual from 'fast-deep-equal';
import {computed, onScopeDispose, toValue, watch, watchEffect} from 'vue';
import {useRoute, useRouter} from 'vue-router';

/**
 * Converts a plural word to a singular word.
 * It's not very clever, so only use it for words that end in 's' or 'es'.
 *
 * @param {string} name
 * @return {string}
 */
function toSingular(name) {
  return name.replace(/s$/, '');
}

/**
 * Capitalise the first letter of each word.
 *
 * @param {string} name
 * @return {string}
 */
function toTitleCase(name) {
  return name.replace(/\b./, str => str.toUpperCase());
}

/**
 * Create a route that can show a single service page of the given category.
 *
 * @param {string} category - one of ServiceNames
 * @param {string?} pathPrefix
 * @param {import('vue-router').RouteLocationRaw?} parent - where to return from the service page
 * @return {import('vue-router').RouteRecordRaw}
 */
export function useServiceRoute(category, pathPrefix = '/', parent = undefined) {
  const single = toSingular(category);
  return {
    name: single,
    path: pathPrefix + single,
    children: useServiceRoutes(category, parent),
    props: () => ({
      parent: parent
    }),
    meta: {
      authentication: {
        rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
      },
      title: toTitleCase(single)
    }
  };
}

/**
 * Create routes that can show a single service page of the given category.
 *
 * @param {string} category - one of ServiceNames
 * @param {import('vue-router').RouteLocationRaw?} parent - where to return from the service page
 * @return {import('vue-router').RouteRecordRaw[]}
 */
export function useServiceRoutes(category, parent = undefined) {
  const jsonEditorRoute = (name) => /** @type {import('vue-router').RouteRecordRaw} */ ({
    path: 'config',
    name,
    component: () => import('@/dynamic/service/ServiceJsonEditor.vue'),
    props: (route) => ({
      name: serviceName(route.params.name, category),
      id: route.params.id,
      parent
    })
  });
  const catchAllRoute = {path: ':rest(.*)', component: PluginPageLoading};

  const categoryName = toSingular(category);
  return [{
    name: categoryName + '-name-id',
    path: ':name/:id',
    components: {
      // PluginParent does the redirection for us based on the service type and plugin info
      default: () => import('./service/ServicePluginParent.vue'),
      nav: () => import('./service/ServicePluginNav.vue')
    },
    props: route => {
      return {
        name: route.params.name,
        id: route.params.id,
        category,
        parent
      };
    },
    children: [jsonEditorRoute(categoryName + '-name-id-json'), catchAllRoute]
  }, {
    name: categoryName + '-id',
    path: ':id',
    component: {
      // PluginParent does the redirection for us based on the service type and plugin info
      default: () => import('./service/ServicePluginParent.vue'),
      nav: () => import('./service/ServicePluginNav.vue')
    },
    props: route => {
      return {
        name: '',
        id: route.params.id,
        category,
        parent
      };
    },
    children: [jsonEditorRoute(categoryName + '-id-json'), catchAllRoute]
  }];
}

/**
 * Create a link (suitable for `to` in a `router-link`) to a service page that will show the given service.
 *
 * @param {MaybeRefOrGetter<string | undefined>} category - One of ServiceNames
 * @param {MaybeRefOrGetter<string | undefined>} name - The name of the device hosting the service
 * @param {MaybeRefOrGetter<string | undefined>} id - The id of the service
 * @return {{
 *   hasLink: Ref<boolean>,
 *   to: Ref<undefined|import('vue-router').RouteLocationRaw>,
 *   toManualEdit: Ref<undefined|import('vue-router').RouteLocationRaw>
 * }}
 */
export function useServiceRouterLink(category, name, id) {
  const router = useRouter();
  const uiConfig = useUiConfigStore();

  // We have a link target whenever we know which service to edit.
  const linkable = computed(() => Boolean(toValue(category) && toValue(id)));
  const to = computed(() => {
    if (!linkable.value) return undefined;
    if (toValue(name)) {
      return {
        name: toSingular(toValue(category)) + '-name-id',
        params: {name: toValue(name), id: toValue(id)}
      };
    } else {
      return {
        name: toSingular(toValue(category)) + '-id',
        params: {id: toValue(id)}
      };
    }
  });
  // hasLink is only true when the link target exists AND the edit page is
  // actually available to the user, i.e. its route path is enabled by the UI
  // config. Without this check we'd show an edit button that navigates to a
  // page the navigation guard immediately redirects away from.
  const hasLink = computed(() => {
    if (!to.value) return false;
    let resolved;
    try {
      resolved = router.resolve(to.value);
    } catch {
      return false;
    }
    if (!resolved.matched.length) return false;
    return uiConfig.pathEnabled(resolved.path);
  });
  const toManualEdit = computed(() => {
    if (!linkable.value) return undefined;
    if (toValue(name)) {
      return {
        name: toSingular(toValue(category)) + '-name-id-json',
        params: {name: toValue(name), id: toValue(id)}
      };
    } else {
      return {
        name: toSingular(toValue(category)) + '-id-json',
        params: {id: toValue(id)}
      };
    }
  });
  return {
    hasLink,
    to,
    toManualEdit
  };
}

/**
 * Like useServiceRouteLink, but uses the context from the current route and sidebar store.
 *
 * @return {{hasLink: Ref<boolean>, to: Ref<import('vue-router').RouteLocationRaw|undefined>}}
 */
export function useSidebarServiceRouterLink() {
  const route = useRoute();
  const sidebar = useSidebarStore();
  return useServiceRouterLink(() => route.meta?.editRoutePrefix, () => sidebar.data?.nodeName, () => sidebar.data?.service?.id);
}

/**
 * @param {import('vue-router').RouteLocation | import('vue-router').RouteLocationMatched | import('vue-router').RouteRecordRaw} route
 * @return {boolean}
 */
export function isServiceRoute(route) {
  return route.name?.endsWith('-id') || route.name?.endsWith('-name-id');
}

/**
 * Configures the routes for a plugin and redirects to the first route configured or the default json editor route.
 *
 * @param {MaybeRefOrGetter<CategoryPlugin | undefined>} plugin
 */
export function usePluginRoutes(plugin) {
  const _p = computed(() => /** @type {CategoryPlugin|undefined} */ toValue(plugin));
  const routes = computed(() => _p.value?.routes);
  const router = useRouter();
  let addedRoutes = /** @type {(() => void)[]} */ [];
  const removeAll = () => {
    for (const r of addedRoutes) {
      r();
    }
    addedRoutes = [];
  };

  const route = useRoute();
  const parentName = computed(() => {
    // the parent should be the route that is showing the ServicePluginParent page
    const r = route.matched.find(r => isServiceRoute(r));
    return r?.name;
  });

  watch([routes, parentName], ([newRoutes, newParent], [oldRoutes, oldParent]) => {
    if (deepEqual(newRoutes, oldRoutes) && newParent === oldParent) return;
    if (oldRoutes) {
      removeAll();
    }
    if (newRoutes) {
      for (const r of newRoutes) {
        if (newParent) addedRoutes.push(router.addRoute(newParent, r));
        else addedRoutes.push(router.addRoute(r));
      }

      // if we are already on a location for a route we just added, we need to reload the page
      router.replace(route)
          .catch((e) => console.warn('Failed to reload page after adding routes', e));
    }
  }, {immediate: true, deep: true});

  // clean up routes when the component is destroyed
  onScopeDispose(() => removeAll());
}

/**
 * Redirects to the first page of a plugin when it is loaded.
 *
 * @param {MaybeRefOrGetter<CategoryPlugin | undefined>} plugin
 * @param {MaybeRefOrGetter<boolean>} loaded
 */
export function usePluginRedirect(plugin, loaded) {
  const router = useRouter();
  const route = useRoute();

  const firstPage = /** @type {import('vue').ComputedRef<import('vue-router').RouteLocationRaw>} */ computed(() => {
    const pr = toValue(plugin)?.routes;
    if (pr && pr.length) {
      const first = pr[0];
      if (first.name) return {name: first.name};
      return {path: route.path + '/' + first.path};
    }
    return {path: route.path + '/config'};
  });
  const shouldRedirect = computed(() => {
    return isServiceRoute(route) && toValue(loaded);
  });
  watchEffect(() => {
    if (shouldRedirect.value) {
      router.replace(firstPage.value)
          .catch(e => console.warn('Failed to redirect to first page of plugin', e));
    }
  });
}
