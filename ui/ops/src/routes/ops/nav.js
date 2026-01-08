import {useUiConfigStore} from '@/stores/uiConfig.js';
import {computed} from 'vue';

/**
 * @typedef {Object} NavItem
 * @property {string} title - The title of the menu item
 * @property {string} icon - The icon to display for the menu item
 * @property {{path: string}} link - The route link for the menu item
 * @property {string} [badgeType] - Optional badge type for the menu item
 * @property {function} [enabled] - Optional function to determine if the item is enabled
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
    title: 'Downloads',
    icon: 'mdi-download',
    link: {path: '/ops/downloads'},
    badgeType: null,
    enabled: () => {
      const uiConfig = useUiConfigStore();
      return uiConfig.pathEnabled('/downloads') && uiConfig.config?.ops?.downloadsServerEndpoint;
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
