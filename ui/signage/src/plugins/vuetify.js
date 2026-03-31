import '@mdi/font/css/materialdesignicons.css';
import {createVuetify} from 'vuetify';
import {Intersect} from 'vuetify/directives';

// a map like {en: 'en-GB', fr: 'fr-FR', ...} used to correctly format dates based on the users preferences
const dateLocales = (navigator?.languages ?? ['en-GB']).reduce((acc, lang) => {
  const parts = lang.split('-');
  if (parts.length > 1 && acc[parts[0]] === undefined) {
    acc[parts[0]] = lang;
  }
  return acc;
}, {});

export default createVuetify({
  directives: {
    Intersect
  },
  date: {
    locale: dateLocales
  },
  theme: {
    defaultTheme: 'dark',
    themes: {
      dark: {
        dark: true,
        colors: {
          'surface': '#282F36', // neutral: for cards and menu content, etc
          'surface-lighten-2': '#585D63',
          'primary-darken-4': '#00525E',
          'primary-darken-1': '#00AAC1',
          'primary': '#00BED6',
          'primary-lighten-1': '#33CADE',
          'primary-lighten-2': '#4DD1E2',
          'primary-lighten-3': '#66D7E6',
          'primary-lighten-4': '#80DEEB',
          'primary-lighten-5': '#99E5EF',
          'secondary-darken-1': '#C5CC3C',
          'secondary': '#DAE343',
          'secondary-lighten-1': '#E2E969',
          'secondary-lighten-2': '#E6EB7B',
          'secondary-lighten-3': '#E9EE8E',
          'secondary-lighten-4': '#EDF1A1',
          'secondary-lighten-5': '#F1F4B4',
          'secondary-teal-darken-1': '#004651',
          'secondary-teal': '#195962',
          'secondary-teal-lighten-1': '#336B74',
          'secondary-teal-lighten-2': '#4D7E85',
          'secondary-teal-lighten-3': '#669097',
          'secondary-teal-lighten-4': '#80A3A8',
          'secondary-teal-lighten-5': '#99B5B9',
          'accent-darken-1': '#FFA400',
          'accent': '#FFAF25',
          'accent-lighten-1': '#FFB833',
          'accent-lighten-2': '#FFC14D',
          'accent-lighten-3': '#FFCA66',
          'accent-lighten-4': '#FFD380',
          'accent-lighten-5': '#FFDB99',
          'neutral-darken-1': '#101820',
          'neutral': '#282F36',
          'neutral-lighten-1': '#40464D',
          'neutral-lighten-2': '#585D63',
          'neutral-lighten-3': '#707479',
          'neutral-lighten-4': '#888C90',
          'neutral-lighten-5': '#9FA3A6',
          'neutral-lighten-6': '#B7BABC',
          'neutral-lighten-7': '#CFD1D2',
          'neutral-lighten-8': '#ECEDED',
          'neutral-lighten-9': '#F7F7F7',
          'error-darken-1': '#BB0434',
          'error': '#D0043C',
          'error-lighten-1': '#D93661',
          'error-lighten-2': '#DE4F75',
          'error-lighten-3': '#E36889',
          'error-lighten-4': '#E8829D',
          'error-lighten-5': '#EC9BB0',
          'success-darken-1': '#00613E',
          'success': '#008052',
          'success-lighten-1': '#00955F',
          'success-lighten-2': '#00A76B',
          'success-lighten-3': '#00B674',
          'success-lighten-4': '#00C27C',
          'success-lighten-5': '#0CCE88',
          'info': '#00BED6',
          'warning': '#FFA400'
        }
      }
    }
  },
  defaults: {
    VContainer: {
      VCard: {
        elevation: 0,
        class: 'rounded-lg'
      },
    },
    VDataTable: {
      hover: true,
      sortAscIcon: 'mdi-arrow-up-drop-circle-outline',
      sortDescIcon: 'mdi-arrow-down-drop-circle-outline'
    },
    VDataTableServer: {
      hover: true,
      sortAscIcon: 'mdi-arrow-up-drop-circle-outline',
      sortDescIcon: 'mdi-arrow-down-drop-circle-outline'
    }
  }
});
