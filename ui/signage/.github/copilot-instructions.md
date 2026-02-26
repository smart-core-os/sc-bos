<!-- Copilot instructions for signage -->

# How to help in this repository

This repo is a small Vue 3 + Vite app used to render a signage-style water tank SVG component. Keep suggestions minimal, concrete, and specific to the files and patterns found here.

- Project entry: `src/main.js` mounts `App.vue` and registers Pinia. Router is present (`src/router/index.js`) but commented out.
- Single purpose components live in `src/components/` (e.g. `WaterTank.vue`) and are used directly by `App.vue`.
- State is small and in Pinia (example: `src/stores/counter.js`) — prefer simple reactive composition API patterns.

What to do first

- Inspect `src/main.js`, `src/App.vue`, and `src/components/WaterTank.vue` to understand UI flow.
- Run locally with the npm scripts in `package.json`:
  - `npm install`
  - `npm run dev` (Vite dev server)

Coding conventions and patterns (explicit, discoverable)

- Vue 3 Composition API with <script setup> is used (see `WaterTank.vue`, `App.vue`). Follow that style for new components.
- Props are defined using `defineProps({ ... })` and are passed directly in templates (e.g. `<WaterComp waterUnit="321" />`).
- Local DOM refs are used to query and mutate SVG content (`ref` + `onMounted` + `svg.value.querySelectorAll(...)`). When editing SVG logic, prefer manipulating transforms/styles rather than changing raw paths at runtime.
- CSS is kept in single-file components with `scoped` styles.
- Pinia stores use the setup-style `defineStore('name', () => { ... })` and export composable stores from `src/stores`.

Build and developer workflow notes

- Scripts: `dev` (vite), `build` (vite build), `preview` (vite preview), `lint` (eslint . --fix), `format` (prettier --write src/).
- Vite resolves `@` to `./src` via `vite.config.js`. Use `@/components/...` for imports when appropriate.

Common edits and examples

- To change the water display logic, edit `src/components/WaterTank.vue`:
  - The displayed value is stored in `currentWaterUnit` and updated by `setValue()` inside `onMounted`.
  - The component demos auto-updating values via an interval; if adding external data sources, replace `startAuto()` with a prop or store subscription and clear intervals on unmount.
- To add routes, enable `router` in `src/main.js` and add route objects in `src/router/index.js`.

Integration & external dependencies

- Dependencies: `vue@3`, `pinia`, `vue-router`. Dev-only: `vite`, ESLint + Prettier, `vite-plugin-vue-devtools`.
- No networked services or APIs are present in the repo. If integrating external data, prefer injecting via Pinia or props and avoid manipulating component internals from outside.

What NOT to change without asking

- Project scaffolding (Vite config, package.json engines) — keep Node engine constraints intact.
- The SVG path data in `WaterTank.vue` unless the visual change is intentional — prefer transform/style tweaks for animation.

Where to look for examples

- `src/components/WaterTank.vue` — complex SVG + runtime DOM manipulation example.
- `src/stores/counter.js` — canonical Pinia store shape used in this project.

When uncertain, ask the maintainer for:

- Intended data source for `WaterTank` (live telemetry vs demo random values).
- Whether router should be enabled (it is present but commented out).

If you edit files, keep diffs small and preserve formatting (Prettier + ESLint configured).

— End
