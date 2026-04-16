# Space UI

A touch-panel UI for Smart Core BOS. Designed to be installed on a wall-mounted tablet in a meeting room or shared space — it connects to a running SC BOS instance, lets occupants control lighting and temperature, and shows the current time and date.

---

## Table of Contents

- [Developer Setup](#developer-setup)
- [Environment Variables](#environment-variables)
- [UI Config (Server-side)](#ui-config-server-side)
- [Authentication](#authentication)
- [Panel Setup Flow (First Run)](#panel-setup-flow-first-run)
- [Panel Modes](#panel-modes)
  - [Single Room](#single-room)
  - [Combined Room](#combined-room)
  - [Admin Mode](#admin-mode)
- [Zone Filtering (Local Accounts)](#zone-filtering-local-accounts)
- [Reconfiguring a Panel](#reconfiguring-a-panel)

---

## Developer Setup

**Prerequisites:** Node 22.x, Yarn 1.22.x, a running SC BOS instance.

```bash
# Install dependencies
cd ui/space
yarn install --frozen-lockfile --check-files

# Start the dev server (http://localhost:5173)
yarn dev

# Production build (output to dist/)
yarn build

# Lint
yarn lint:nofix   # check only
yarn lint         # auto-fix
```

The dev server proxies gRPC-Web requests to the SC BOS backend. Point it at your instance using an `.env` file (see below).

---

## Environment Variables

Create a `.env.local` file in `ui/space/` (or copy and edit `.env.vanti-ugs`):

| Variable | Description | Example |
|---|---|---|
| `VITE_GRPC_ENDPOINT` | Base URL of the SC BOS gRPC-Web endpoint | `https://localhost:8443` |
| `VITE_UI_CONFIG_URL` | URL of the `ui-config.json` file served by SC BOS | `https://localhost:8443/__/my-site/space-ui.json` |
| `VITE_AUTH_URL` | Base URL for local auth token endpoint (defaults to same origin) | `https://localhost:8443` |

If `VITE_UI_CONFIG_URL` is not set, the UI fetches `{BASE_URL}ui-config.json` from its own origin.

---

## UI Config (Server-side)

The SC BOS server serves a `ui-config.json` file that the space UI fetches on every page load. It controls widgets, authentication, and theming. If the file is unreachable, built-in defaults are used.

### Minimal example

```json
{
  "config": {
    "auth": {
      "disabled": true
    },
    "pages": {
      "home": {
        "widgets": [
          {"name": "lighting"},
          {"name": "temperature"}
        ]
      }
    },
    "theme": {
      "logoUrl": "/img/powered-by.svg"
    }
  }
}
```

### Widgets

The `config.pages.home.widgets` array controls which cards appear on the home screen. Available widgets:

| Name | Description |
|---|---|
| `lighting` | Brightness slider with auto/manual mode toggle |
| `temperature` | Temperature set-point slider with auto/manual HVAC toggle |
| `air-quality` | Air quality score indicator |

Widgets are shown in the order they appear in the array. If `widgets` is omitted, no controls are shown.

### Theme

```json
{
  "config": {
    "theme": {
      "logoUrl": "/img/powered-by.svg"
    }
  }
}
```

`logoUrl` is the image shown at the bottom of the home screen. Defaults to `img/powered-by.svg` from the UI's public directory.

### Authentication config

See [Authentication](#authentication) below.

---

## Authentication

Authentication is optional. By default the UI will attempt to use a local account login if no auth config is provided.

### Disable authentication

```json
{
  "config": {
    "auth": {
      "disabled": true
    }
  }
}
```

When disabled, no login screen is shown and all zones are accessible.

### Local accounts (username/password)

Local accounts are managed in SC BOS. The UI posts credentials to `{VITE_AUTH_URL}/oauth2/token` using the password grant flow and decodes the returned JWT to get claims.

No additional `ui-config.json` settings are needed to enable local auth — it is on by default.

### Keycloak (SSO)

```json
{
  "config": {
    "auth": {
      "keycloak": {
        "realm": "Smart_Core",
        "url": "https://keycloak.example.com/",
        "clientId": "scos-spaceui"
      }
    }
  }
}
```

### Selecting providers

If multiple providers are configured, the login page offers a choice. To restrict which providers are shown:

```json
{
  "config": {
    "auth": {
      "providers": ["localAuth"]
    }
  }
}
```

Valid values: `"localAuth"`, `"keyCloakAuth"`, `"deviceFlow"`.

---

## Panel Setup Flow (First Run)

On first load (or after a reset), the UI runs a short setup wizard before reaching the home screen:

1. **Login** — if authentication is enabled, the user logs in.
2. **Select your room** — the user picks a zone from a list. This zone is saved to browser storage and used every time the panel loads.
3. **Select combined room** *(optional)* — the user can optionally select a second zone to create a "combined room". This can be skipped.
4. **Home screen** — the panel is now configured and shows the zone controls.

The setup wizard re-appears automatically if the panel has never been configured, or when a reconfigure is triggered (see [Reconfiguring a Panel](#reconfiguring-a-panel)).

There is also a 5-minute idle timeout during setup. If no interaction occurs the panel returns to the login screen (or home, if auth is disabled).

---

## Panel Modes

Once set up, the panel operates in one of three modes. The mode is saved in browser storage and persists across reloads.

### Single Room

**The default mode.** The panel is assigned to one specific zone chosen during setup. Every time it loads it goes straight to that room's controls — no selection needed.

The home screen shows:
- Current time and date
- The room name (below the date)
- Configured widgets (lighting, temperature, etc.)
- A "Switch to Combined Room" button (only if a combined room was configured)

### Combined Room

When two adjacent rooms are booked together, a "combined room" zone can be configured during setup. The home screen shows a toggle button to switch between the two views.

- **Single view** — shows controls for the panel's own room.
- **Combined view** — shows controls for the joined zone.

The toggle persists across reloads. If a combined room was not configured during setup, the toggle is not shown.

### Admin Mode

Admin mode is for a roving panel or shared tablet that should not be locked to a single room. Instead of saving a zone, the panel shows the full zone list on every visit. The admin selects a zone, adjusts controls, and can return to the list without anything being permanently saved.

**Enabling Admin Mode during setup:**

On the "Select your room" screen, tap **Admin Mode** (below the zone list) instead of selecting a room.

**Using Admin Mode:**

1. The panel shows the zone list with the current time.
2. Tap a zone to open its controls (lighting and temperature).
3. Tap the back arrow to return to the zone list.
4. Changes to lighting/temperature take effect immediately — they are not buffered.
5. No zone is ever saved to browser storage.

**Zone list filtering still applies** — if the logged-in account has zone restrictions, only permitted zones appear in the list.

**To reconfigure from Admin Mode:** use the same 10-click escape (see below).

---

## Zone Filtering (Local Accounts)

When a user logs in with a local account, the SC BOS server can include a `zones` claim in the JWT. If present, the setup zone list (and the admin mode zone list) only shows zones that match or fall under those paths.

Example JWT claim:
```json
{
  "zones": ["bhx-19c/floors/02/zones"]
}
```

This will show `bhx-19c/floors/02/zones` and all of its sub-zones (e.g. `bhx-19c/floors/02/zones/meeting-room-03`) but hide zones on other floors.

If `zones` is absent from the token the user sees all zones.

---

## Reconfiguring a Panel

To return a panel to the setup wizard (for example, to change the room or switch modes):

**Hidden trigger:** Tap the clock on the home screen (or admin mode zone list) **10 times** within one second. After 5 taps a countdown is shown. On the 10th tap, the panel clears its configuration and returns to setup.

If authentication is enabled, the user will be asked to log in again before setup runs.

To cancel a reconfigure mid-flow, tap **Return to home** on the login screen.
