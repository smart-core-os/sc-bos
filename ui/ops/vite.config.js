import vue from '@vitejs/plugin-vue';
import {execSync} from 'child_process';
import {globSync} from 'glob';
import {createRequire} from 'module';
import {dirname, posix, relative, sep} from 'path';
import {fileURLToPath, URL} from 'url';
import {defineConfig, loadEnv} from 'vite';
import eslintPlugin from 'vite-plugin-eslint';
import vuetify from 'vite-plugin-vuetify';
import svgLoader from 'vite-svg-loader';

const _require = createRequire(import.meta.url);

// Any import that resolves to a local filesystem dependency that isn't an ESM dependency needs to be in this list.
const optimizeDepsInclude = [
  '@smart-core-os/sc-bos-panzoom-package' // cjs imports that use the yarn workspace also need to be in this list
];
// Typically that includes local proto files that are referenced via either `file:../` dependencies in package.json
// or via yarn/npm linking the generated sources into this project (or both).
// This snippet will find all *_pb.js files and ensure that they will be handled correctly by vite.
for (const dep of ['@smart-core-os/sc-bos-ui-gen', '@smart-core-os/sc-api-grpc-web']) {
  // find proto files in projects
  const protoDirRoot = dirname(_require.resolve(dep + '/package.json'));
  const globPattern = posix.join(protoDirRoot, '!(node_modules)/**/*_pb.js');
  const protoFiles = globSync(globPattern)
      .map(p => dep + '/' + relative(protoDirRoot, p).replaceAll(sep, posix.sep));
  optimizeDepsInclude.push(...protoFiles);
  // remove the .js extension so import statements without .js still use the bundle
  optimizeDepsInclude.push(...protoFiles.map(f => f.substring(0, f.length - 3)));
}

const gitCommand = 'git describe --tags --always --match ui/*';

// https://vitejs.dev/config/
export default defineConfig(({mode}) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    define: {
      GIT_VERSION: JSON.stringify(env.GIT_VERSION || execSync(gitCommand).toString().trim())
    },
    optimizeDeps: {
      include: optimizeDepsInclude,
      // See https://github.com/vueuse/vue-demi
      exclude: ['vue-demi']
    },
    build: {
      commonjsOptions: {
        // This should include regexes for any directory that should be processed by the commonjs transform.
        // The entries match against filesystem paths (not import paths) and resolve after symbolic links.
        include: [/node_modules/, /ui-gen/, /panzoom-package/]
      }
    },
    plugins: [
      vue(),
      vuetify({
        styles: {
          configFile: 'src/sass/settings.scss'
        }
      }),
      svgLoader(),
      eslintPlugin()
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    }
  };
});
