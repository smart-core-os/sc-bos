import {fileURLToPath, URL} from 'url';
import {defineConfig} from 'vitest/config';

// Deliberately standalone rather than merged with vite.config.js: that config shells out to
// `git describe` and loads the vue/vuetify/svg plugins, none of which unit tests need.
// Add plugins here only when a test actually requires them (e.g. mounting a .vue component).
export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  test: {
    include: ['src/**/*.spec.js']
  }
});
