# Smart Core Ops UI

This ui allows you to setup and manage smart core nodes (area controllers, app servers, edge gateways, etc). This is the
default ui you get when you visit any of these boxes IPs.

It's located here in the project because we don't yet know how much of it will be 'Smart Core' and how much will be
'Project', so best to be specific and pull out the commonality when we actually have proof it'll be needed.

## Getting started

Run the UI in dev mode (hot reloading, etc) with `yarn run dev` - assuming you have run `yarn install` and have the
dependencies ready. You will likely also need to `yarn install` in [../ui-gen](../ui-gen) as this package depends on
that file-based package.

The OPS UI is currently used for both commissioning and management tasks, viewing device information and dashboards.
The plan is to migrate the operation and dashboard elements to Smart Core Connect.

The design
follows [this Figma design](https://www.figma.com/proto/5wfaoD7k13k1g0XTbdoc3q/SmartCore-Design-System-v1.0?page-id=420%3A2128&node-id=495%3A2440&viewport=202%2C130%2C0.32&scaling=min-zoom&starting-point-node-id=420%3A5995).

The different features of the UI require different capabilities from the server, here are some examples:

- Logging in with keycloak requires keycloak to be running - see [docs/install/](../../docs/install/dev.md)
- Logging in with username + password requires an area controller configured with local user accounts -
  see [area-controller/README.md](../../cmd/area-controller/README.md#local-authentication) for details
- Notifications requires a controller configured with the [alert system](../../pkg/system/alerts/README.md)
- System pages, like lights, require - or at least work better - with some actual devices to view and manipulate. These
  can be configured on the controller via the `"drivers"` config, samples of which can be found
  in [config/samples](../../config/samples). For detailed config options see the `internal/driver` sub-packages, each of
  which should provide a readme and config structure.

## Development Setup (Recommended)

The easiest way to get a full development environment with mock devices is:

1. **Start the backend** using the JetBrains run configuration `.run/BOS [23557,8443] (UGS).run.xml`
   
   Or run directly from the command line:
   ```bash
   go run ./cmd/bos --policy-mode check --data .data/vanti-ugs --appconf example/config/vanti-ugs/app.conf.json --sysconf example/config/vanti-ugs/system.conf.json
   ```
   
   - This runs SC-BOS with the Vanti UGS example config from `example/config/vanti-ugs/`
   - Includes mock devices for lighting, HVAC, sensors, meters, and shades across multiple floors
   - Listens on ports 23557 and 8443

2. **Start the UI** in dev mode with the Vanti UGS environment:
   ```bash
   cd ui/ops
   yarn run dev --mode vanti-ugs
   ```
   - This uses the `.env.vanti-ugs` configuration which points to `https://localhost:8443`
   - Provides hot module reloading for rapid development

3. **Add more devices** as needed:
   - Use the Ops UI's device management features to add mock devices interactively
   - Or edit `example/config/vanti-ugs/app.conf.json` to add more driver configurations

This setup gives you a complete, working environment with realistic mock data for testing all UI features.

## Alternative Setup

If you need to connect to a different local area controller, create a `.env.local` file in this directory with:
```properties
VITE_GRPC_ENDPOINT=https://localhost:23557
```

