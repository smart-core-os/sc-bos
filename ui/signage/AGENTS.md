# signage

A Vue 3 + Vuetify app for building signage screens: a widget dashboard plus full-screen views for water, energy,
occupancy, air quality and temperature. It reads live and historical data from sc-bos over gRPC-web.

## Running

From `ui/` (it is a Yarn 1 workspace; don't use npm):

```bash
yarn --cwd signage dev --mode=vanti-ugs   # against a local controller on https://localhost:8443
yarn --cwd signage lint:nofix
```

`--mode=vanti-ugs` loads `.env.vanti-ugs`, which points at a controller running the
`example/config/vanti-ugs` config and logs in as the dashboard user.

## Structure

- `src/router/index.js`: one route per view. Each route's props come from the UI config (`props`, `props.water`,
  `props.energy`, ...), which says which device or zone names the view shows.
- `src/stores/ui-config.js`: loads the UI config from `VITE_UI_CONFIG_URL` (for the example,
  `example/config/vanti-ugs/signage.json`), falling back to built-in defaults.
- `src/traits/<trait>/`: composables that expose each trait's data to the views, built on the gRPC-web calls in
  `src/api/sc/traits/` (which wrap the generated clients from `@smart-core-os/sc-bos-ui-gen`).
- `src/views/`: one component per route. `src/components/`: the visual pieces (`WaterTank.vue`, meters, ...).

## Conventions

- Composition API with `<script setup>`; Pinia setup stores (`defineStore('name', () => {...})`).
- A view gets the names it displays from route props, never hard-coded.
- In the SVG components, animate with transforms and styles rather than rewriting path data at runtime.
