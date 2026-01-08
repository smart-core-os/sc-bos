import SidebarPage from '@/components/pages/SidebarPage.vue';
import downloads from '@/routes/ops/downloads/route.js';
import notifications from '@/routes/ops/notifications/route.js';
import overview from '@/routes/ops/overview/route.js';
import securityEvents from '@/routes/ops/security-events/route.js';
import waste from '@/routes/ops/waste/route.js';

import {route} from '@/util/router.js';

export default {
  name: 'ops',
  path: '/ops',
  redirect: '/ops/loading',
  components: {
    default: SidebarPage,
    nav: () => import('./OpsNav.vue')
  },
  children: [
    {
      path: 'loading',
      component: () => import('./OpsLoading.vue')
    },
    ...route(overview),
    ...route(downloads),
    {
      path: 'emergency-lighting',
      component: () => import('./emergency-lighting/EmergencyLighting.vue'),
      meta: {
        authentication: {
          rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
        }
      }
    },
    {
      path: 'security',
      component: () => import('./security/SecurityHome.vue'),
      meta: {
        authentication: {
          rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
        }
      }
    },
    {
      path: 'air-quality',
      component: () => import('./air-quality/AirQuality.vue'),
      meta: {
        authentication: {
          rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
        }
      }
    },
    ...route(notifications),
    ...route(securityEvents),
    ...route(waste),
  ],
  meta: {
    authentication: {
      rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
    },
    title: 'Operations'
  }
};
