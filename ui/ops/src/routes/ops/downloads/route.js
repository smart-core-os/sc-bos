export default {
  path: 'downloads',
  components: {
    default: () => import('./DownloadFileCard.vue'),
  },
  meta: {
    title: 'Downloads',
    authentication: {
      rolesRequired: ['superAdmin', 'admin', 'commissioner', 'operator', 'viewer']
    }
  }
};
