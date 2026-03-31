import auth from '@/routes/auth/route.js';
import automations from '@/routes/automations/route.js';
import devices from '@/routes/devices/route.js';
import ops from '@/routes/ops/route.js';
import system from '@/routes/system/route.js';
import {useAccountStore} from '@/stores/account';
import {useSidebarStore} from '@/stores/sidebar';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {route, routeTitle} from '@/util/router.js';

import {nextTick} from 'vue';
import {createRouter, createWebHistory} from 'vue-router';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('./login/LoginPage.vue'),
      meta: {authentication: {rolesRequired: false}, title: 'Login'}
    },
    ...route(auth),
    ...route(devices),
    ...route(ops),
    ...route(automations),
    ...route(system)
  ]
});

if (window) {
  router.beforeEach(async (to, from, next) => {
    const uiConfig = useUiConfigStore();
    await uiConfig.loadConfig();
    const authDisabled = uiConfig.auth.disabled;
    const accountStore = useAccountStore();
    // Initialize Local and Keycloak auth instances,
    // so we can check if the user is logged in and/or manage the login flow
    try {
      await accountStore.initialise(uiConfig.config?.auth?.providers);
    } catch (e) {
      console.error('Failed to initialize the account store', e);
    }

    // ------------------------ Data store logic ------------------------ //

    const sidebar = useSidebarStore();

    // Reset the sidebar to defaults if the active route changes
    if ((to.name && to.name !== from.name) ||
        (!to.name && to.path !== from.path)) {
      sidebar.resetSidebarToDefaults();
    }

    // ------------------------ NavigationGuard logic ------------------------ //
    /**
     * Navigation guard for handling route access based on authentication status and application configuration(s).
     *
     * Step 1. Check if the requested path is enabled in the config. If it's not enabled
     *         and the user is trying to access a path other than '/login', redirect to the home path.
     *
     * Step 2. If authentication is not disabled (authDisabled is false):
     *    a. If the user is not logged in:
     *       i.   If the user is trying to access a non-login path, store the current path in session storage
     *            for redirecting back after login, and then redirect the user to the '/login' page.
     *    b. If the user is logged in:
     *       i.   If the user is trying to access the '/login' path, redirect them to the home path.
     *       ii.  If the user is navigating to a regular page (not the '/login' page), check for a saved
     *            redirect path in session storage. If a saved redirect path exists, remove it from session
     *            storage and redirect the user to that path.
     *
     * Finally, if none of the above conditions are met, proceed to the next route (allow the navigation).
     */
    const isPathEnabled = uiConfig.pathEnabled(to.path);
    const redirectToHome = () => {
      const home = uiConfig.homePath;
      // If homePath doesn't resolve to a known route, it would cause an infinite redirect loop.
      // Fall back to the default home path instead.
      const resolved = router.resolve(home);
      next(resolved.matched.length > 0 ? home : uiConfig.defaultConfig.config.home);
    };
    const isLoginPath = to.path === '/login';
    const isAuthenticated = accountStore.isLoggedIn;

    if (!isPathEnabled && (!isLoginPath || authDisabled && isLoginPath && from.path !== uiConfig.homePath)) {
      redirectToHome();
      return;
    }

    if (!authDisabled) {
      if (!isAuthenticated) {
        if (to.path !== '/login') {
          // Store the current path to redirect back after login
          window.sessionStorage.setItem('redirect', to.fullPath);
          next('/login');
          return;
        }
      } else {
        if (to.path === '/login') {
          // Redirect logged-in users away from the login page to home
          redirectToHome();
          return;
        } else {
          // If navigating to a regular page, check for a saved redirect path
          const savedRedirect = window.sessionStorage.getItem('redirect');
          if (savedRedirect) {
            window.sessionStorage.removeItem('redirect');
            next(savedRedirect); // Redirect to the saved path
            return;
          }
        }
      }
    }

    next();
  });

  router.afterEach((to, from) => {
    const nt = routeTitle(to);
    const ot = routeTitle(from);
    if (nt === ot) {
      return;
    }
    const titleParts = [];
    const prefix = import.meta.env.VITE_PAGE_TITLE_PREFIX || '';
    if (prefix) {
      titleParts.push(prefix);
    }
    if (nt) {
      titleParts.push(nt);
    }
    const suffix = import.meta.env.VITE_PAGE_TITLE_SUFFIX || 'Smart Core';
    if (suffix) {
      titleParts.push(suffix);
    }
    const title = titleParts.join(' - ');
    // noinspection JSIgnoredPromiseFromCall
    nextTick(() => window.document.title = title);
  });
}

export default router;
