# Smart Core Feature Overview

## How to read this document

Make sure you understand [the concepts](#sc-concepts) first, they are mentioned throughout the document.

It's really hard to describe all that Smart Core does, things are generally interconnected and rely on each other.
Concepts are shared and reused between different areas. There are a lot of moving parts.

The document is structured around being able to discover what smart core can and can't do. If it's not listed there's a
good change it's not supported. Check the heading and see if anything matches what you're looking for.

Any changes we've thought of but haven't done should be listed with the following difficulty ratings:

- TRIVIAL - changes that aren't even worth mentioning, changing a 1 to a 2, adding a new line to a list, etc.
- EASY - the change can typically be copy-pasted from another part of the code, no new concepts need designing.
- MODERATE - there will need to be some thought and design work but the platform is flexible enough to accommodate the
  change.
- HARD - the change will require significant design work and typically require changes to the platform.
- EXTREME - the change breaks a number of assumptions present and relied upon. Many things will need to change to
  accommodate this change.

## SC concepts

### SC BOS vs SC API

SC API (or sc-api) describes the published API of Smart Core. We describe our API using `.proto` files which say things
like _Brightness exists and has a field levelPercent with decimal type_ and _there is a service called LightApi with a
method GetBrightness_.

The api is just a description of how two machines/apps can talk with one another, it doesn't actually exist anywhere at
this level. It's more of a specification for communication and a shared understanding of concepts.

SC BOS implements the spec defined by sc-api. It provides a server that clients can ask the _GetBrightness_ question of
amongst a number of other questions that aren't defined by sc-api. To implement this api, sc-bos introduces a number of
new concepts to model devices, automations, zones and others.

SC BOS has defined some APIs locally that should eventually be published to sc-api, once they've proven they work and
cover the cases needed.

### Traits

Traits describe how you interact with a device, can you adjust the brightness or see if it's open/closed.
The concept of a trait is defined by sc-api, the core Smart Core API.
Check out [the Smart Core guide](https://smart-core-os.github.io/guide/traits.html) for more information on traits. See
[below](#traits-supported-by-sc-bos) for a list of traits supported by SC BOS.

Traits are made up of a number of different aspects:

- The control API, where you turn a light on or subscribe to occupancy changes.
- The info API, where you can find out what unit a meter reads or which lighting presets exist.
- The history API, where you can ask for past values from a particular trait.

Traits are modelled around resources, the Light trait has the Brightness resource, the Electric trait has the Demand
resource (amongst others).

Traits do not exist in a binary form: exist or not exist, there is a scale.

- Has the trait been defined? Does there exist a description of what the trait is called, what it's for, what resources
  it has, what API does it have.
- Has the trait been published? Does it exist as part of sc-api or is it sc-bos specific or project specific. You can
  only say Smart Core supports trait Foo if it's published.
- Does the trait define the info aspect?
- Does the trait define the history aspect?
- Is there a mock implementation of the trait?
- Do zones support the trait?
- Does the Ops UI have widgets that can display the trait information?
- Do any drivers implement this trait?

Each of these things needs work to complete, they don't come for free and different traits might have different
combinations of the above.
The features implemented for a trait are usually driven by what has been asked for.

Adding new features to a trait is usually an EASY change, creating a new trait is MODERATE. Publishing a trait to sc-api
is also MODERATE.

#### Trait History

The history API was invented as part of sc-bos and hasn't been published to sc-api or the guide yet.

History is stored in a general purpose postgres database, either directly or via the [Hub](#hub-nodes-and-enrolment).
History records can have a TTL in both time and number of records. History is not recorded by default, we are not sure
on the resource requirements for this, especially with the use of the general purpose database instead of a time-series
db.

Enabling history for traits that support it is TRIVIAL, adding history support to a trait is EASY. There is no query
language or aggregation for historical records, if you want to see meter readings for the last month we have to read
every record for that time period and do the aggregation on the front end.

Enabling history for all traits and devices is HARD and will likely involve the introduction of a time-series database.
Enabling aggregate queries and efficient access to historical data is also HARD.

History is implemented in sc-bos via an automation, the automation subscribes to changes in the trait and writes the
changes to storage.

### Zones

Zones are a way to group devices together and treat them as a single entity.
For example you can define a new zone "Room 2" and associate a bunch of lights, air temperature, occupancy sensors, and
other device traits with it. Then you can say something like "is Room 2 occupied" or "turn off the lights in Room 2" and
the correct thing will happen.

Zones are a feature that was invented for sc-bos, sc-api and the rest of Smart Core doesn't know about them.

We've implemented zones as a collection of features, a feature is like 'lighting' or 'occupancy'. Zones can have many
features based on their config. Here are all the supported features:

- Access
- Air Quality
- Electric
- Enter Leave
- HVAC - Air Temperature trait support
- Lighting - Light trait support
- Meter
- Mode
- Occupancy - including converting an Enter Leave trait into occupancy
- Open Close
- Status

Adding new features is MODERATE, to add a new feature we have to design how grouping works for that trait type. For
example when we added the lighting feature we had to decide what brightness level the zone had if each light in the zone
reported a different brightness.

Zones have a name just like any other device in Smart Core, the name is how you identify what you are interacting with
via the API. Most features support a sub-name concept where you can sub-divide the devices in a zone into specialist
groups. An example of this is defining a zone's meters to be the total meter value, but define sub-named for lighting,
hvac, and power.

The Ops UI has no special zone support, zones are treated like any other device and appear on the devices page or can
be use as the source for dashboard widgets.

Zones are displayed prominently when configuring tenant access to the SC BOS API, we pre-select "show only zones" during
the permissions step in this UI. You can deselect this to show all device names if you desire.

Zones are configured via a JSON file stored on the SC BOS machine, the config describes the name and features of the
zone, listing, for example, the list of light names or thermostat names.

Adjusting the devices in a zone or adding new zones are EASY changes for a dev to make, it currently requires knowledge
of the zone config JSON structure. There is an outstanding task to enable user editable config that removes this
restrictions, but it is still in the design phase. Fixing this is HARD.

### Automations

Automations are independent chunks of code that we've written to do something automatically with SC BOS data.
Examples include changing the mode of FCUs to "Unoccupied" based on Occupancy Sensor data from PIRs, or periodically
sending an email to a known address with meter readings for a month.

See [automations supported by SC BOS](#automations-supported-by-sc-bos) for details of specific automations.

Each automation has been hand crafted by a developer to do the task required of it, there is no automation builder or
if-this-then-that style engine in SC BOS (though we want to add one eventually, but that's HARD).

Multiple instances of an automation can be setup and run at the same time with different config. For example some sites
have 100s of lighting automation instances running monitoring and adjusting lighting levels for different groups of
lights.

Each automation, once written, is configured via a JSON file stored on the SC BOS machine. The config describes the
adjustable aspects of the automation: PIR timeout, names of lights to control, which mode means "occupied", etc.
These config files are not user editable, they are typically edited by a developer and then deployed to the SC BOS
machines. As with zones, allowing user editable config is HARD and in the pre-design phase.

### Authentication and Authorization

As a refresher: authentication (authn) is about confirming who you are, authorization (authz) is about allowing you to
perform some action.

In Smart Core authn is defined by OAuth2, the industry standard for authn. OAuth2 in general is a token based authn
mechanism, i.e. there exists some process for you to get a token and if you give your token to Smart Core then SC will
trust you are who you say you are. This isn't intended to be an OAuth course so we won't go into too many details here,
suffice it to say OAuth calls these different ways to get a token Grants or Flows and Smart Core supports a few of them
in different ways:

#### Password Grant

You give Smart Core your username and password, SC validates them against some data store, and if successful issues you
a token. This flow is simple enough that sc-bos implements it directly, no third party needed. The account store is a
JSON file on the local disk of the sc-bos server which contains your username and a hashed password along with claims
used for the authz process.

There is no user centric way to edit the contents of this JSON file.

We don't recommend that this auth method is used as it is less secure and doesn't support single sign on or other
advanced features like revocation.

The Ops UI and client specific UIs support this and call it "Local Login."

#### Client Credentials Grant

Very similar to the password grant, you exchange a client id and client secret for an access token. This flow is
typically used for machine to machine communication, i.e. for tenant API access. Once again, sc-bos implements this flow
directly.

Accounts for this flow can be stored in two places, a local JSON file or a Postgres database. The JSON file is for fixed
API access, typically for developer testing. The Ops UI has admin pages for managing the postgres accounts.

#### Keycloak

Keycloak (KC) is a third party authentication server, widely used in the industry. If you've ever heard of AD FS, Auth0,
or
have worked with cloud provider IAM or auth services you're in the same area. We rely on Keycloak to provide additional
OAuth grant flows for Smart Core.

Technically sc-bos only really needs to know the server is an OpenID Connect server, but we've never tested with
anything other than KC and there are likely assumptions we've made that don't translate to other authn providers. Adding
new providers is likely EASY, though might be MODERATE depending on those assumptions.

##### KC Authentication Flow

This is the most common user flow for browser apps. You click login, you are redirected to a login page and shown a
"grant this app some permissions" confirmation box, then redirected back to the application to continue your work.

KC has been setup on multiple sites to use both Active Directory stores and the KC local database (postgres) for storing
account data.

The Ops UI and project specific UIs support this and call it "Keycloak Login."

##### KC Device Authentication Flow

Useful for apps that have limited user input, like a signage screen, this flow shows you a QR code to scan and a
sequence of letters you should enter using another device in order to log in the app. Just like you get when logging
into Netflix on your TV.

The account you log in using is the same as with the other KC flows, you just do it on your personal device instead of
on the hardware hosting the limited input app.

The Ops UI and project specific UIs support this and call it "Use your device to log in". It would be unusual to use
this flow on the Ops UI, though it is available.

#### Authorization

At a high level, authorization involves comparing who you are (the claims in your token) with what your are trying to
do (your request) and making sure they match. SC BOS implements this using policy files, files that contain rules like
_allow write if token is present and valid and the "roles" claim includes "admin"_. There are a lot of authz policies
we've written, some are pattern driven: _api requests that start with "Update" are writes_, some are specific: _only
the "admin" role can access the "tenant management" API_.

SC BOS uses open policy agent (OPA) to write and enforce these policies, OPA is an open source library and language
designed for this exact purpose.

Right now all policies have been written by developers and are embedded into each sc-bos deployment. There is no user
editing of these policy files (and likely shouldn't be). Specific project deployments can replace the policies if they
need to.

Our policy files authorise based on these criteria:

1. APIs are guarded by the roles claim. Tenant edits needs the admin role, write access needs more than the viewer role,
   etc. Roles do not limit which devices you can access, only what you can do with them.
2. Devices are guarded by the zones claim. If your zones claim includes the name `floors/2` then you can access any
   device called `floors/2/*` and do anything with those devices.
3. Some APIs are public, typically discovery and system health APIs. You can always ask "is SC healthy", even when not
   authenticated.

While users can have both roles and zones, we have no policies that combine these claims. Our policies are written in
an "allow list" type of way, if you are in the allow list you can do perform the action, you get on the allow list if
your role matches OR your zone matches.

### Hub, nodes, and enrolment

In a deployment where there are multiple SC instances (area controller, building controller, etc), getting those nodes
talking to each other requires some coordination. The hub is where this coordination happens. Nodes are enrolled with
the hub, which updates the node to trust communication from any other enrolled nodes.

The hub keeps track of which nodes are enrolled and can return this list to those that ask, this enables features like
edge gateway and proxies along with Ops UI features like SC health and central management of devices.

Enrolment is a manual one-off process performed on the hub to add new nodes to it. You can do this via the Ops UI.

A node running with `--policy-mode=on` only lets a hub inspect and enrol it if the hub presents the node's setup code.
A new node issues a setup code only if it has local accounts on, no accounts yet, and no other way to log in (such as
Keycloak or an identities file). It logs where to read the code, usually `<data>/setup/setup-code.txt`. A node with
policy on that issues no code can't be enrolled by a hub.

Pass the code as `setup_code` in `InspectHubNode` and `EnrollHubNode`, or to `cmd/tools/enrollment-tool` with
`--setup-code`; the Ops UI enrol dialog does not ask for it yet. Inspecting a node with a code needs the same access on
the hub as enrolling one, because every wrong code uses up one of the node's limited guesses. The hub sends the code to
the node and never stores it. Once the node is enrolled or has an account, it stops accepting the code. Nodes listed in
the hub's `nodes` config are enrolled without a code, so those need policy `check` or `off`.

## Traits supported by SC BOS

### Access

Events relating to access control, a card scanner or QR code reader, and the outcome of that scan.

No sc-bos devices implement Access.

These project specific integrations implement Access:

- AxiomXa - access control system

| Feature | Notes                                                       |
|---------|-------------------------------------------------------------|
| Zones   | Not supported                                               |
| History | Not supported                                               |
| Mock    | Supported, updates automatically                            |
| Ops UI  | Table cell, sidebar card. Can be integrated with Open Close |
| Docs    | Not published, local to sc-bos                              |

### Air Quality Sensor

We can model the following properties
([see details](https://smart-core-os.github.io/api/traits/air_quality_sensor.html#traits-air-quality-sensor-proto-2)):

- CO2 level
- VOCs
- Air Pressure
- Comfort Index
- Infection Risk
- Quality Score
- PM1, PM2.5, PM10
- Air Change Rate

Additional Air Quality metrics are EASY to add.

These devices report Air Quality:

- AirThings Sensors
- Steinel HPD Sensors

| Feature | Notes                              |
|---------|------------------------------------|
| Zones   | Supported, direct and grouped      |
| History | Supported, not default             |
| Mock    | Supported, updates automatically   |
| Ops UI  | Sidebar card, ops page with graphs |
| Docs    | Published, as part of sc-api       |

### Air Temperature

We can model Air Temperature - aka thermostats. Typically linked with FCUs and other space heating/cooling systems.
Supports set point control and ambient temperature and humidity.
([see details](https://smart-core-os.github.io/api/traits/air_temperature.html#traits-air-temperature-proto-2))

These devices report Air Temperature:

- AirThings Sensors
- BACnet Devices
- Shelly TRV Devices
- Steinel HPD Sensors

| Feature | Notes                                      |
|---------|--------------------------------------------|
| Zones   | Supported, direct and grouped              |
| History | Supported, not default                     |
| Mock    | Supported, static value                    |
| Ops UI  | Table cell, sidebar card, dashboard widget |
| Docs    | Published, as part of sc-api               |

### Brightness Sensor

Measures the measured brightness of a space, typically a function of PIRs
([see details](https://smart-core-os.github.io/api/traits/brightness_sensor.html#traits-brightness-sensor-proto-2)).

No sc-bos devices implement Brightness Sensor.

These project specific integrations implement Brightness Sensor:

- DALI (via Beckhoff TC3) - linked to PIRs
- ZenControl (via MQTT) - linked to PIRs

| Feature | Notes                        |
|---------|------------------------------|
| Zones   | Not supported                |
| History | Not supported                |
| Mock    | Not supported                |
| Ops UI  | Not supported                |
| Docs    | Published, as part of sc-api |

### Button

Models a button, typically a physical button on a wall or a virtual button in an app.

No sc-bos devices implement Button.

These project specific integrations implement Button:

- DALI (via Beckhoff TC3) - for dali transitive buttons, aka light switches

| Feature | Notes                          |
|---------|--------------------------------|
| Zones   | Not supported                  |
| History | Not supported                  |
| Mock    | Not supported                  |
| Ops UI  | Not supported                  |
| Docs    | Not published, local to sc-bos |

### Color

Models the color of a light, for example to set the color of an RGB light.

No sc-bos devices implement Color.

These project specific integrations implement Color:

- Architainment structural lighting

| Feature | Notes                          |
|---------|--------------------------------|
| Zones   | Not supported                  |
| History | Not supported                  |
| Mock    | Not supported                  |
| Ops UI  | Not supported                  |
| Docs    | Not published, local to sc-bos |

### Electric

Models live electrical use, typically from a meter or sub-meter, can report active power use and more niche power use
like reactive power and power factor
([see details](https://smart-core-os.github.io/api/traits/electric.html#traits-electric-proto-2)).
We don't have anything in production that supports the 'mode' feature of this trait.

These devices implement Electric:

- BACnet Devices

| Feature | Notes                                     |
|---------|-------------------------------------------|
| Zones   | Supported, direct and grouped             |
| History | Supported, not default                    |
| Mock    | Supported, updates automatically          |
| Ops UI  | Table cell, sidebar card, dashboard graph |
| Docs    | Published, as part of sc-api              |

### Energy Storage

Models the storage of energy, just like a battery
([see details](https://smart-core-os.github.io/api/traits/energy_storage.html#traits-energy-storage-proto-2)).

These devices implement Energy Storage:

- AirThings Sensors battery level

| Feature | Notes                        |
|---------|------------------------------|
| Zones   | Not supported                |
| History | Not supported                |
| Mock    | Supported, static value      |
| Ops UI  | Not supported                |
| Docs    | Published, as part of sc-api |

### Enter Leave Sensor

Tracks people entering and leaving a space, typically as they cross a line or enter a zone
([see details](https://smart-core-os.github.io/api/traits/enter_leave_sensor.html#traits-enter-leave-sensor-proto-2)).

These devices implement Enter Leave:

- Xovis Space Sensors - line crossing only

Also implemented by these project specific integrations:

- AxiomXa - access control system

| Feature | Notes                         |
|---------|-------------------------------|
| Zones   | Supported, direct and grouped |
| History | Not supported                 |
| Mock    | Supported, static value       |
| Ops UI  | Table cell, sidebar card      |
| Docs    | Published, as part of sc-api  |

### Emergency

Represents the status of an emergency signal, like a fire alarm
([see details](https://smart-core-os.github.io/api/traits/emergency.html#traits-emergency-proto-2)).

This has not been deployed to any site due to third party issues.

These devices implement Emergency:

- BACnet Devices

These project specific integrations also implement Emergency:

- TC3 IO - contact closures linked to fire alarms

| Feature | Notes                        |
|---------|------------------------------|
| Zones   | Not supported                |
| History | Not supported                |
| Mock    | Not supported                |
| Ops UI  | Not supported                |
| Docs    | Published, as part of sc-api |

### Fan Speed

Models the speed of a fan, for example to turn off the fan on an FCU
([see details](https://smart-core-os.github.io/api/traits/fan_speed.html#traits-fan-speed-proto-2)).

These devices implement Fan Speed:

- BACnet Devices

| Feature | Notes                        |
|---------|------------------------------|
| Zones   | Not supported                |
| History | Not supported                |
| Mock    | Supported, fixed value       |
| Ops UI  | Not supported                |
| Docs    | Published, as part of sc-api |

### Light

Control and telemetry for the brightness of a light
([see details](https://smart-core-os.github.io/api/traits/light.html#traits-light-proto-2)).

These devices implement Light:

- Wiser KNX

Also implemented by these project specific integrations:

- DALI (via Beckhoff TC3)
- ZenControl (via MQTT)

| Feature | Notes                                                    |
|---------|----------------------------------------------------------|
| Zones   | Supported, direct and grouped. Supports read-only lights |
| History | Not supported                                            |
| Mock    | Supported, fixed value                                   |
| Ops UI  | Table cell, sidebar card, control                        |
| Docs    | Published, as part of sc-api                             |

### Light Test

A special purpose trait that allows executing emergency lighting tests and reporting the results.

No sc-bos devices implement Light Test.

These project specific integrations implement Light Test:

- DALI (via Beckhoff TC3)

| Feature | Notes                                    |
|---------|------------------------------------------|
| Zones   | Not supported                            |
| History | Special, supported via dedicated storage |
| Mock    | Not supported                            |
| Ops UI  | Dedicated Emergency Lighting page        |
| Docs    | Not published, local to sc-bos           |

### Meter

Models the consumption of a resource, like litres of water or kWh of electricity.

These devices implement Meter:

- BACnet Devices

| Feature | Notes                                     |
|---------|-------------------------------------------|
| Zones   | Supported, direct and grouped             |
| History | Supported, not default                    |
| Mock    | Supported, updates automatically          |
| Ops UI  | Table cell, sidebar card, dashboard graph |
| Docs    | Not published, local to sc-bos            |

### Mode

Allows setting a flag to one of a set of modes, for example to set a building to 'holiday' mode
([see details](https://smart-core-os.github.io/api/traits/mode.html)).

Usually used for toggles and settings both for our own automations and for controlling devices.

These devices implement Mode:

- BACnet Devices - can be linked to Binary or Analog values
- The Settings Driver - used to store reusable modes, like "lighting.mode = 'maintenance'"

Also implemented by these project specific integrations:

- Architainment structural lighting - for setting predefined sequences

| Feature | Notes                                                                    |
|---------|--------------------------------------------------------------------------|
| Zones   | Supported, direct and grouped. Modes can be adjusted as part of the zone |
| History | Not supported                                                            |
| Mock    | Supported, static value                                                  |
| Ops UI  | Sidebar card                                                             |
| Docs    | Published, as part of sc-api. Described by the info api.                 |

### Occupancy Sensor

Reports the occupancy of a space, normally implemented by a PIR sensor, can also report people count
([see details](https://smart-core-os.github.io/api/traits/occupancy_sensor.html#traits-occupancy-sensor-proto-2)).

These devices implement Occupancy Sensor:

- PestSense Driver
- Steinel HPD Sensors
- Xovis Space Sensors

Also implemented by these project specific integrations:

- DALI (via Beckhoff TC3) - linked to PIRs
- ZenControl (via MQTT) - linked to PIRs

| Feature | Notes                                                                                     |
|---------|-------------------------------------------------------------------------------------------|
| Zones   | Supported, direct and grouped. Enter Leave sensors can be converted to occupancy sensors. |
| History | Supported, not default                                                                    |
| Mock    | Supported, updates automatically                                                          |
| Ops UI  | Table cell, sidebar card, dashboard graph                                                 |
| Docs    | Published, as part of sc-api.                                                             |

### Open Close

Models things that can open and close, like a door or a window
([see details](https://smart-core-os.github.io/api/traits/open_close.html#traits-open-close-proto-2)).

No sc-bos devices implement Open Close.

These project specific integrations implement Open Close:

- AxiomXa - door status
- Axis IO Devices - contact closures linked to doors

| Feature | Notes                            |
|---------|----------------------------------|
| Zones   | Not supported                    |
| History | Not supported                    |
| Mock    | Supported, updates automatically |
| Ops UI  | Table cell, sidebar card         |
| Docs    | Published, as part of sc-api.    |

### On Off

A simple trait for turning something on or off, usually a power socket but sometimes a light
([see details](https://smart-core-os.github.io/api/traits/on_off.html#traits-on-off-proto-2)).

No sc-bos devices implement On Off.

These project specific integrations implement On Off:

- Architainment structural lighting - for turning the lights on / off

| Feature | Notes                         |
|---------|-------------------------------|
| Zones   | Not supported                 |
| History | Not supported                 |
| Mock    | Supported, static value       |
| Ops UI  | Not supported                 |
| Docs    | Published, as part of sc-api. |

### PTZ - Pan Tilt Zoom

Controls the pan, tilt, and zoom of a camera
([see details](https://smart-core-os.github.io/api/traits/ptz.html#traits-ptz-proto-2)).

No sc-bos devices implement PTZ.

These project specific integrations implement PTZ:

- Hikvision - for controlling cameras

### Status

Reports the status (health) of a device, for example if a sensor is offline.
After deployment to a couple of sites, and after some feedback, we realised this trait does not model the interactions
people were expecting.
People wanted some way to see all the equipment they need to do something with: fix, call an engineer about, things that
affect safety or compliance. This does not do that very well.

There are plans to introduce more specific traits for modelling "someone needs to check on this device" and "SC can't
talk to this device", which are both bundled into the Status trait currently.

These devices implement Status:

- AirThings Sensors
- BACnet Devices

These project specific integrations also implement Status:

- AxiomXa - for device/comm health
- DALI (via Beckhoff TC3) - for device status and emergency lighting test results
- Hikvision - for camera status
- ZenControl - for comm status and emergency light test results

| Feature | Notes                                         |
|---------|-----------------------------------------------|
| Zones   | Supported, direct                             |
| History | Indirectly supported via the Alerts mechanism |
| Mock    | Supported, static value                       |
| Ops UI  | Table cell, sidebar card, Notifications page  |
| Docs    | Not published, local to sc-bos                |

### UDMI

A special purpose trait that allows a driver to publish messages in UDMI format to an MQTT broker. This trait exists
because time was tight and we needed some way to do this without modelling all properties as traits.

As the structure and UDMI conformance is handled by individual drivers, the payloads can be a little inconsistent.

These devices implement UDMI:

- BACnet Devices - configured per site, but typically all points are published
- Steinel HPD Sensors

These project specific integrations also implement UDMI:

- AxiomXa access control - door status and access events
- DALI (via Beckhoff TC3) - lighting, pirs, and emergency lighting
- Hikvision - camera status and events
- ZenControl - lighting, pirs, and emergency lighting

| Feature | Notes                                         |
|---------|-----------------------------------------------|
| Zones   | Not supported                                 |
| History | Indirectly supported via the Alerts mechanism |
| Mock    | Supported, static value                       |
| Ops UI  | Table cell, sidebar card, Notifications page  |
| Docs    | Not published, local to sc-bos                |

###

## Automations supported by SC BOS

### Azure IoT

This auto publishes trait resources to Azure IoT Hub. Written for a client data dashboard project originally, it has
been written without that specific infrastructure in mind.

The automation subscribes to changes in a configured list of device traits, serialises the changes to JSON, and
publishes them to a configured MQTT topic.

The auto supports exporting these traits:

- Air Quality Sensor
- Air Temperature
- Brightness Sensor
- Light
- Meter
- Occupancy Sensor

### BMS

This auto performs a few automated tasks for BMS systems.

1. It subscribes to occupancy changes and sets a mode on FCUs to "occupied" or "unoccupied" based on the occupancy
   state.
2. It updates a mode on FCUs between "normal" and "eco" depending on the time of day and day of week.
3. It adjusts the default set point of FCUs based on the outdoor air temperature using a rolling mean algorithm.

All things the auto subscribes to and things it adjusts can be configured. The time of day and day of week are
configured via a cron-like string.

### History

The auto subscribes to changes in a configured list of device traits and writes the changes to some storage.

Supported storage engines:

- Direct to postgres db
- Via the hub
- Local BoltDB db
- To a specific history storage server (via API)
- Keep history in memory

### Lights

This auto performs automated adjustment of lights based on different inputs. The responsibilities are:

1. Turn lights on when PIRs are triggered.
2. Toggle lights when a button is pressed.
3. Turn lights on or off when a button is pressed.
4. Turn lights off after some period of inactivity.
5. Adjust the brightness of lights based on the ambient light level.

The configuration of timeouts and brightness can be changed at runtime via either a schedule or a mode change on some
other device. For example switching the mode to "maintenance" can adjust the minimum brightness to 50% and maximum to
100% to quickly see both functional automation and any broken lights.

This automation is quite flaky due to the complexity of all the different features working together.

### Meter Mail

This auto sends a templated email containing meter readings to a configured email address at a configured time. The
automation feeds into operators meter and billing processes.

### Occupancy Email

This auto sends weekly reports of historical occupancy (people counts) to a configured email address. The automation was
created to support occupancy planning, for example ordering or preparing food to match expected occupation.

### Reset Enter Leave

A simple automation that resets the enter leave counts at specific times of day. The time of day is configured via a
cron-like string.

### Status Alerts

This auto subscribes to all status trait instances and records changes in the alerts database table, which ultimately
show up on the Ops UIs notification page.

### Status Email

This auto sends periodic digest emails to a configured email address with changes in the status of all devices. This is
like an emailed version of the notifications page on the Ops UI.

### UDMI

This auto subscribes to all UDMI trait instances and publishes the changes to a configured MQTT server.

## Drivers supported by SC BOS

Drivers link external systems with Smart Core. At their core they implement traits backed by third party system
interaction.

### Air Things

Exposes Air Things IAQ cloud API via Smart Core traits. We integrate with their sample data api, but also support
filtering based on location.

### BACnet

General support for exposing BACnet objects as Smart Core traits.

The driver is configured in two steps, first you define the BACnet side of things, what comm settings, which devices,
which objects and properties exist and what to call them. Secondly you combine those points together into device traits.

BACnet systems can be quite varied in their implementation, even on one site. The architecture of this driver allows us
to support all these variations.

The BACnet driver uses a third party gobacnet library which is not feature complete, notably it lacks support for COV.
All subscription in the BACnet driver are done via polling. We have forked and made many changes to this underlying
library to support new features or fix bugs we've come across during projects.

### Mock

The mock driver allows for easier testing by providing a way to simulate devices and their traits. Typically not used in
production, the developers use it extensively to develop new features in sc-bos and the Ops UI without needing a real
site to test against.

Some traits are implemented as simple static data holders, some have automatic updates to values to make them more
realistic.

### PestSense

A one-off driver written for the Facilities show in 2022, it integrates with a rat-trap device and can report occupancy
of the trap.

### Proxy

An internal use driver that allows Smart Core devices defined on other nodes to appear as if they were defined on this
node. This is useful for automations that need local access to devices, and also for landlord/tenant setups where the
tenant wants to treat landlord devices as if they were their own.

### SE / Wiser KNX

Connects the Schneider Electric Wiser KNX system to Smart Core.

### Shelly

Connects the Shelly TRVs to Smart Core.

### Steinel HPD

Connects the Steinel HPD sensors to Smart Core. These are presence and multi-sensor devices which we expose over
different traits.

### Xovis

Exposes Xovis presence detector to Smart Core, we only support the line crossing feature of the Xovis sensor and expose
it as Enter Leave.

## SC Ops UI

The Ops UI is a web application that comes bundled with (most) sc-bos installations. We deploy it at the node level and
at the global level - to show a unified view. Depending on which node you're looking at, you'll see different devices
and features.

Deploying the Ops UI doesn't come for free, though setting it up per site/node is EASY.

Features of the Ops UI are enabled and configured via a dedicated UI Config file which lists things like which pages to
display, how auth is configured, where to fetch floor plans, and so on. UI config is not user editable and is written
and deployed by a developer.

In general the Ops UI is divided into 4 main categories:

- Operations - daily use pages like the dashboards or the notification page
- Devices - unstructured view of everything the building has in it
- Admin - setting up tenants, checking status
- Internals - hub configuration, stopping/starting automations, and that type of thing

### Operations

Where normal Ops UI users should be spending most of their time, this is where the _suitable panes of glass_ concept is
implemented. Here you'll find pages for security or air quality, notifications or emergency lighting, as well as
overview dashboards.

#### Dashboards

Also called the overview pages, these collect a number of different metrics about the building or zones and show them
via dedicated widgets. From energy charts, to temperature gauges.

Widgets on the dashboards are made by developers and configured to be used on the dashboard. Dashboard pages and
sub-pages are flexible in layout and widget combinations. They are set up via the UI Config JSON file which is written
by a developer and deployed to each node. There are plans to add user customisable dashboards, but this is HARD and
needs design.

#### Notifications

A view into the alerts database, this shows a remembered list of all the status values for each device on site with
limited filtering capabilities. You can download a CSV for the notifications from this page.

You can see past notifications for any particular device by clicking on it.

The notifications page has the concept of acknowledgement, someone can click a button to say they've _seen it_.

Notifications that resolve themselves are hidden from the table automatically.

#### Air Quality

A page for looking at air quality sensor data and history for any air quality sensor in the building. You select your
sensor and are shows a graph with the readings from that sensor.

#### Security

A page that shows devices in the `acs` subsystem as both cards and on a hand-crafted floor plan. This shows door
status (open, closed) and access attempts: card read successful. We show the last person to attempt to access the
reader.

#### Emergency Lighting

A view of all emergency lights in the building with their latest test status. You can download the table as a CSV and
select specific lights to run functional or duration tests.

### Devices

A filterable and searchable table of each named device configured in sc-bos. Each row of the table represents a single
name and we show pertinent data about the traits that device implements in line. For example we show the brightness of
lights, the occupancy of PIRs, the latest reading of meters, etc.

You can filter the devices by subsystem, floor or zone. You can search for specific devices by name or other metadata.

Clicking a device shows more details about the traits the device implements and allows interaction with that device if
possible, for example change the set point or adjust the mode.

You can download a CSV of the current or historical data for the filtered devices on the device table.

### Admin

The Ops UI allows users to create and manage tenant accounts and credentials through a dedicated page. You can change
tenant zones and issue or revoke secrets.

### Internals

#### Workflows and automations

This page shows all the automations configured for the node or site. If looking at a gateway you can select which node
to view automations for.

Each automation has a name and status: running or stopped. If the automation stopped unexpectedly you can see a reason
in the form of an error message. You can stop and start automations from here.

#### System / Drivers

This page is just like the automations page, but shows drivers. Drivers are responsible for creating most of the devices
you see in the devices page.

#### System / Components

A page that shows all the nodes enrolled with our hub and some summary information about their drivers and automations.

You can enrol new nodes and forget existing nodes from this page.
