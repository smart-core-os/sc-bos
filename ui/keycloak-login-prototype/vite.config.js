import vue from '@vitejs/plugin-vue'

import {globSync} from 'glob';
import {createRequire} from 'module';
import {dirname, join, relative} from 'path';
import {fileURLToPath, URL} from 'url'

import {defineConfig} from 'vite'

const _require = createRequire(import.meta.url);

// Any import that resolves to a local filesystem dependency that isn't an ESM dependency needs to be in this list.
const optimizeDepsInclude = [];
// Typically that includes local proto files that are referenced via either `file:../` dependencies in package.json
// or via yarn/npm linking the generated sources into this project (or both).
// This snippet will find all *_pb.js files and ensure that they will be handled correctly by vite.
for (const dep of ['@smart-core-os/sc-bos-ui-gen']) {
  // find proto files in projects
  const protoDirRoot = dirname(_require.resolve(dep + '/package.json'));
  const protoFiles = globSync(join(protoDirRoot, '!(node_modules)/**/*_pb.js'))
      .map(p => dep + '/' + relative(protoDirRoot, p));
  optimizeDepsInclude.push(...protoFiles);
  // remove the .js extension so import statements without .js still use the bundle
  optimizeDepsInclude.push(...protoFiles.map(f => f.substring(0, f.length - 3)));
}

// https://vitejs.dev/config/
export default defineConfig({
  optimizeDeps: {
    include: optimizeDepsInclude
  },
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    commonjsOptions: {
      include: [/@ew-auth-poc\/ui-gen/, /node_modules/]
    }
  }
})
