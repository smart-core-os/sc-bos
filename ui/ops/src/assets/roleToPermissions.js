/**
 * This module exports a mapping between different user roles and their respective permissions
 * in terms of the access levels (full, limited, or blocked) they have to different routes in the application.
 *
 * The route paths are represented as regex patterns to facilitate matching against absolute route paths.
 *
 * Each role object contains three properties:
 * - fullAccess: an array of routes that the role has full access to.
 * - limitedAccess: an array of routes that the role has limited access to.
 * - blockedAccess: an array of routes that the role has no access to.
 *
 * The roles defined are as follows:
 * - admin: Has the highest level of access with full access to all of the routes and no blocked routes.
 * - superAdmin: Similar to the admin but tailored for supervisory roles, with extensive full access routes.
 * - commissioner: A role with considerable access but restricted from accessing authentication-related routes.
 * - operator: A role with a mixture of full and limited access to various routes, ideal for operation-related tasks.
 * - signage: Primarily has access to signage related routes, with all other routes being blocked.
 * - viewer: A role with full access to viewing signage and limited access to all other routes.
 *
 * @type {Object<string, {fullAccess: string[], limitedAccess: string[], blockedAccess: string[]}>}
 */
export const roleToPermissions = {
  admin: {
    fullAccess: [
      '/auth',
      '/auth/users',
      '/auth/third-party',
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components',
      '/system/logs',
      '/signage'
    ],
    limitedAccess: [],
    blockedAccess: []
  },
  commissioner: {
    fullAccess: [
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components',
      '/signage'
    ],
    limitedAccess: [],
    blockedAccess: [
      '/auth',
      '/auth/users',
      '/auth/third-party',
      '/system/logs'
    ]
  },
  operator: {
    fullAccess: [
      '/auth',
      '/auth/third-party',
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/signage'
    ],
    limitedAccess: [
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components'
    ],
    blockedAccess: [
      '/auth/users',
      '/system/logs'
    ]
  },
  signage: {
    fullAccess: [
      '/signage'
    ],
    limitedAccess: [],
    blockedAccess: [
      '/auth',
      '/auth/users',
      '/auth/third-party',
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components',
      '/system/logs'
    ]
  },
  superAdmin: {
    fullAccess: [
      '/auth',
      '/auth/users',
      '/auth/third-party',
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components',
      '/system/logs',
      '/signage'
    ],
    limitedAccess: [],
    blockedAccess: []
  },
  viewer: {
    fullAccess: [
      '/signage'
    ],
    limitedAccess: [
      '/auth',
      '/auth/users',
      '/auth/third-party',
      '/devices',
      '/devices/.*',
      '/ops',
      '/ops/overview',
      '/ops/overview/building',
      '/ops/overview/building/.*',
      '/ops/emergency-lighting',
      '/ops/air-quality',
      '/ops/notifications',
      '/ops/security',
      '/site',
      '/site/zone',
      '/site/zone/.*',
      '/automations',
      '/automations/.*',
      '/system',
      '/system/drivers',
      '/system/features',
      '/system/components'
    ],
    blockedAccess: [
      '/system/logs'
    ]
  }
};
