## NABERS dashboard

Shows air quality data and metrics for NABERS reporting. The rating inputs — net
lettable area, postcode, rated hours, meter names, design targets and off-axis
scenarios — all come from `dashboards-config.json`, so the same build serves any
site. `dashboards-config.json` in this directory is a template: replace the
placeholder values with the ones from the building's Design for Performance
assessment.

### Credentials, and what that means

The dashboard authenticates itself. It reads a username and password from
`VITE_DASHBOARD_USERNAME`/`VITE_DASHBOARD_PASSWORD`, falling back to
`config.username`/`config.password` in `dashboards-config.json`, and exchanges
them for a token via the OAuth2 password grant (`src/stores/auth.js`).

**Both of those places are readable by anyone who can load the page.** Vite bakes
env vars into the bundle at build time, and `dashboards-config.json` is fetched
over plain HTTP by the browser. There is no way to hide a credential in a
single-page app, so this is not a bug to fix in the client — but it does mean:

- Give the dashboard its **own service account**, never a person's login and
  never an administrator's. It only ever issues reads, so scope it to read.
- Treat that account as public. Anyone who can reach the display, or the URL, can
  extract it and call the API directly with the same rights.
- Do not commit real credentials. The `dashboards-config.json` in this directory
  is a template and the values in it are placeholders.

If the deployment can mint a short-lived token server-side and serve it to the
page instead, prefer that; the client already treats the token as opaque and
re-fetches it on `UNAUTHENTICATED`, so only `fetchToken` would change.

### Linking out to the ops UI

The base building breakdown widget can carry a link to a fuller energy view, so
detail lives in the ops UI rather than being crammed onto this screen. Set
`nabersBaseBuilding.opsUrl` (and optionally `opsLinkLabel`) to the ops page to
link to; leave them out and no link is drawn.

## Display

Built as a fullscreen dashboard: the pages size to the viewport (`100vh`) and the
rows are flex, so the layout fills whatever display it is put on. There are no
breakpoints and type sizes are fixed, so it is intended for a large screen rather
than a phone, but no particular resolution is assumed.
