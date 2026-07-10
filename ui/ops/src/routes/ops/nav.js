import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed} from 'vue';

/**
 * @typedef {Object} NavItem
 * @property {string} title - The title of the menu item
 * @property {string} icon - The icon to display for the menu item
 * @property {{path: string}} link - The route link for the menu item
 * @property {boolean} [exact] - Optional exact match for the menu item
 * @property {string} [badgeType] - Optional badge type for the menu item
 * @property {function} [enabled] - Optional function to determine if the item is enabled
 */

/**
 * @typedef {Object} ExternalNavItem
 * @property {string} title - The title of the menu item
 * @property {string} icon - The icon to display for the menu item
 * @property {string} href - The external URL the item links to
 * @property {'_self'|'_blank'} [target] - Where to open the link, defaults to '_blank'
 */

/**
 * Menu Items
 * This is the main list of items
 *
 * @type {NavItem[]}
 */
export const navItems = [
  {
    title: 'Notifications',
    icon: 'mdi-bell-outline',
    link: {path: '/ops/notifications'},
    badgeType: 'unacknowledgedAlertCount',
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/notifications') && (uiConfig.config?.ops?.notifications ?? true);
    }
  },
  {
    title: 'Air Quality',
    icon: 'mdi-air-filter',
    link: {path: '/ops/air-quality'},
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/air-quality') && (uiConfig.config?.ops?.airQuality ?? true);
    }
  },
  {
    title: 'Emergency Lighting',
    icon: 'mdi-alarm-light-outline',
    link: {path: '/ops/emergency-lighting'},
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/emergency-lighting') && (uiConfig.config?.ops?.emergencyLighting ?? true);
    }
  },
  {
    title: 'Security',
    icon: 'mdi-shield-key',
    link: {path: '/ops/security'},
    exact: true,
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/security') && (uiConfig.config?.ops?.security ?? true);
    }
  },
  {
    title: 'Security Events',
    icon: 'mdi-shield-alert',
    link: {path: '/ops/security-events'},
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/security-events') && uiConfig.config?.securityEventsSource;
    }
  },
  {
    title: 'Waste Records',
    icon: 'mdi-recycle',
    link: {path: '/ops/waste'},
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/ops/waste') && uiConfig.config?.ops?.waste;
    }
  }
];

/**
 * Check if a route is enabled.
 *
 * @param {NavItem} item
 * @return {boolean}
 */
export function isRouteEnabled(item) {
  const uiConfig = useUiConfigStore();
  if (!uiConfig.pathEnabled(item.link.path)) return false;
  if (typeof item.enabled === 'function') return item.enabled();
  return true;
}

/**
 * Get a computed reference to the enabled navigation items.
 *
 * @return {ComputedRef<NavItem[]>}
 */
export function useEnabledNavItems() {
  return computed(() => navItems.filter(item => isRouteEnabled(item)));
}

/**
 * Get a computed reference to the external navigation links defined in the UI config.
 *
 * These are deployment-specific links to other applications served alongside the Ops UI
 * (e.g. standalone SPAs on the same origin). They are not router paths, so they are gated
 * purely by their presence in config rather than by pathEnabled().
 *
 * @return {ComputedRef<ExternalNavItem[]>}
 */
export function useExternalNavItems() {
  const uiConfig = useUiConfigStore();
  return computed(() => {
    const items = uiConfig.getOrDefault('ops.externalLinks', []);
    if (!Array.isArray(items)) return [];
    return items.filter(item => item && typeof item.href === 'string' && item.href);
  });
}
