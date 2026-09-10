import App from '@/App.vue';
import pinia from '@/plugins/pinia.js';
import vuetify from '@/plugins/vuetify.js';
import router from '@/routes/router.js';
import {startClock} from '@/util/clock.js';
import '@/main.scss';
import {createApp} from 'vue';

// Started here, for the life of the app, rather than per page: every route reads
// the clock through a store, and a page that forgot to start it would silently
// get the frozen behaviour back. Nothing stops it, which is correct for a
// dashboard intended to stay open. See util/clock.js.
startClock();

createApp(App)
    .use(pinia)
    .use(router)
    .use(vuetify)
    .mount('#app');
