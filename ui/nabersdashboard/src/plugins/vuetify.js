import '@mdi/font/css/materialdesignicons.css';
import {createVuetify} from 'vuetify';

export default createVuetify({
  theme: {
    defaultTheme: 'dark',
    themes: {
      dark: {
        dark: true,
        colors: {
          'background':       '#0C0921',
          'surface':          '#0C0921',
          'primary':          '#7F00FF',
          'secondary':        '#E100FF',
          'error':            '#f89c9b',
          'warning':          '#f7a12c',
          'success':          '#4caf50',
          'success-lighten-1':'#81c784',
          'error-lighten-1':  '#f89c9b'
        }
      }
    }
  }
});
