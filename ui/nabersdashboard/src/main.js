import App from '@/App.vue';
import pinia from '@/plugins/pinia.js';
import vuetify from '@/plugins/vuetify.js';
import router from '@/routes/router.js';
import '@/main.scss';
import {createApp} from 'vue';

createApp(App)
    .use(pinia)
    .use(router)
    .use(vuetify)
    .mount('#app');
