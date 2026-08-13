## NABERS dashboard

Shows air quality data and metrics for NABERS reporting. The rating inputs — net
lettable area, postcode, rated hours, meter names, design targets and off-axis
scenarios — all come from `dashboards-config.json`, so the same build serves any
site. `dashboards-config.json` in this directory is a template: replace the
placeholder values with the ones from the building's Design for Performance
assessment.

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
