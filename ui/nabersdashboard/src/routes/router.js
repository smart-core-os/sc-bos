import HomePage     from '@/routes/home/HomePage.vue';
import LandlordPage from '@/routes/landlord/LandlordPage.vue';
import {createRouter, createWebHistory} from 'vue-router';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path:      '/',
      name:      'home',
      component: HomePage
    },
    {
      path:      '/landlord',
      name:      'landlord',
      component: LandlordPage
    },
    {
      path:     '/:pathMatch(.*)*',
      redirect: '/'
    }
  ]
});

export default router;
