import App from '@/App.vue';
import pinia from '@/plugins/pinia.js';
import vuetify from '@/plugins/vuetify.js';
import router from '@/router/index.js';
import './assets/main.scss';
import {createApp} from 'vue';


createApp(App)
  .use(pinia)
  .use(router)
  .use(vuetify)
  .mount('#app');
