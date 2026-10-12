Running on FreeBSD
==================

In this document: how to run BOS and the Supervisor as FreeBSD services, install the first release, and move an
existing BOS install under the Supervisor. See [`state-model.md`](state-model.md) for how the Supervisor applies
updates, and [`release-artefact.md`](release-artefact.md) for the release format.

On FreeBSD, BOS runs as a native `rc(8)` service under `daemon(8)`, as the unprivileged `sc-bos` user. The
Supervisor runs as root. It installs each release into its own folder under a releases root, points the `current`
link at it, and restarts BOS with `service sc_bos restart`. The BOS service always runs
`<root>/current/bin/sc-bos` and serves `<root>/current/ui/ops`, so its rc script and config never change between
versions.

Reference files
---------------

`supervisor/dist/freebsd/` holds reference files. Copy them to the host and adapt them; sc-bos doesn't install them
itself.

| File                                      | Install to                                              | Purpose                        |
|-------------------------------------------|---------------------------------------------------------|--------------------------------|
| `rc.d/sc_bos_supervisor`                  | `/usr/local/etc/rc.d/sc_bos_supervisor` (`0555`)        | Supervisor service             |
| `rc.d/sc_bos`                             | `/usr/local/etc/rc.d/sc_bos` (`0555`)                   | BOS service                    |
| `newsyslog.conf.d/sc-bos-supervisor.conf` | `/usr/local/etc/newsyslog.conf.d/` (`0644`)             | Supervisor log rotation        |
| `newsyslog.conf.d/sc-bos.conf`            | `/usr/local/etc/newsyslog.conf.d/` (`0644`)             | BOS log rotation               |
| `supervisor-config.json`                  | `/usr/local/etc/sc-bos-supervisor/config.json` (`0644`) | Supervisor config              |
| `system.conf.json.example`                | merge into `/usr/local/etc/sc-bos/system.conf.json`     | BOS settings this layout needs |

`supervisor-config.json` writes out every FreeBSD default, so you can see them. Only `socketGroup` differs from the
built-in defaults. The Supervisor rejects the file if that group doesn't exist, so create the `sc-bos` group first.

`system.conf.json.example` serves HTTPS on port 8443, enables the Supervisor integration, and serves the Ops UI from
the current release. It has no `socketPath`, because BOS's default on FreeBSD matches the Supervisor's.

### rc variables

Set these in `/etc/rc.conf`, for example with `sysrc`.

| Variable                   | Default                                                                                                                |
|----------------------------|------------------------------------------------------------------------------------------------------------------------|
| `sc_bos_supervisor_enable` | `NO`                                                                                                                   |
| `sc_bos_supervisor_config` | `/usr/local/etc/sc-bos-supervisor/config.json`                                                                         |
| `sc_bos_enable`            | `NO`                                                                                                                   |
| `sc_bos_user`              | `sc-bos`                                                                                                               |
| `sc_bos_group`             | `sc-bos`                                                                                                               |
| `sc_bos_root`              | `/usr/local/sc-bos/releases`. Must match the Supervisor's `freebsdrc.root`. Installs delete anything else in it.       |
| `sc_bos_log`               | `/var/log/sc-bos.log`                                                                                                  |
| `sc_bos_flags`             | `--data /var/db/sc-bos --sysconf /usr/local/etc/sc-bos/system.conf.json --appconf /usr/local/etc/sc-bos/app.conf.json` |

`sc_bos_enable` must be `YES` even if you start BOS some other way. Otherwise `service sc_bos restart` prints a
warning and exits successfully without restarting BOS, so every update fails as not confirmed.

`daemon(8)` runs with `-r`, so it starts each service again whenever it exits. BOS relies on this: it exits with
code 0 to restart itself after it downloads a cloud config deployment. Each pidfile holds the pid of `daemon`, not
of the process it runs, so `service ... stop` stops `daemon` and the process together.

Host layout
-----------

| Path                                           | Owner:group     | Mode                  | Contents                                                      |
|------------------------------------------------|-----------------|-----------------------|---------------------------------------------------------------|
| `/usr/local/sbin/sc-bos-supervisor`            | `root:wheel`    | `0755`                | Supervisor binary                                             |
| `/usr/local/etc/sc-bos-supervisor/config.json` | `root:wheel`    | `0644`                | Supervisor config                                             |
| `/var/db/sc-bos-supervisor/`                   | `root:wheel`    | `0700`                | `state.json`, `staging/`                                      |
| `/var/run/sc-bos-supervisor/supervisor.sock`   | `root:sc-bos`   | `0660`                | Supervisor API (folder `0755`)                                |
| `/var/run/sc_bos/`                             | `sc-bos:sc-bos` | `0755`                | BOS service pidfile                                           |
| `/usr/local/sc-bos/releases/`                  | `root:wheel`    | `0755`                | Release folders, `current` and `previous` links, `.lock`      |
| `/usr/local/etc/sc-bos/`                       | `root:sc-bos`   | `0750` (files `0640`) | `system.conf.json`, `app.conf.json`, `ui-config/`             |
| `/var/db/sc-bos/`                              | `sc-bos:sc-bos` | `0750`                | BOS data, including cloud config deployments and registration |
| `/var/log/sc-bos.log`                          | `sc-bos:sc-bos` | `0640`                | BOS log                                                       |
| `/var/log/sc-bos-supervisor.log`               | `root:wheel`    | `0640`                | Supervisor log                                                |

BOS can read its code and config but can't change them. It writes only its data folder and its log. Cloud config
deployments, the cloud registration and local config edits all live in the data folder.

The Supervisor creates its state folder, its socket folder and the releases root when it first needs them. The rc
scripts create the log files and the BOS pidfile folder. You create the rest.

### Listening ports

On FreeBSD, BOS serves HTTPS on port 8443, set with `"listenHttps": ":8443"` in `system.conf.json`. Its default,
port 443, needs root, and BOS runs as the unprivileged `sc-bos` user. gRPC stays on its default port, 23557.

Fresh install
-------------

Run these steps as root on the host, apart from the builds.

1. Create the BOS user and group:

   ```sh
   pw groupadd sc-bos
   pw useradd sc-bos -g sc-bos -d /nonexistent -s /usr/sbin/nologin -c "Smart Core BOS"
   ```

2. Create the BOS config and data folders:

   ```sh
   install -d -o root -g sc-bos -m 750 /usr/local/etc/sc-bos
   install -d -o sc-bos -g sc-bos -m 750 /var/db/sc-bos
   ```

3. Build the Supervisor on any machine with Go, and copy it to the host. Set `GOARCH` to the host's architecture,
   `amd64` or `arm64`:

   ```sh
   CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -trimpath -o sc-bos-supervisor ./supervisor/cmd/supervisor
   ```

4. Install the Supervisor binary, its config, its rc script and its newsyslog config, from a copy of
   `supervisor/dist/freebsd/`:

   ```sh
   install -o root -g wheel -m 755 sc-bos-supervisor /usr/local/sbin/sc-bos-supervisor
   install -d -o root -g wheel -m 755 /usr/local/etc/sc-bos-supervisor
   install -o root -g wheel -m 644 supervisor-config.json /usr/local/etc/sc-bos-supervisor/config.json
   install -o root -g wheel -m 555 rc.d/sc_bos_supervisor /usr/local/etc/rc.d/
   install -d -o root -g wheel -m 755 /usr/local/etc/newsyslog.conf.d
   install -o root -g wheel -m 644 newsyslog.conf.d/sc-bos-supervisor.conf /usr/local/etc/newsyslog.conf.d/
   ```

5. Install the BOS rc script and its newsyslog config:

   ```sh
   install -o root -g wheel -m 555 rc.d/sc_bos /usr/local/etc/rc.d/
   install -o root -g wheel -m 644 newsyslog.conf.d/sc-bos.conf /usr/local/etc/newsyslog.conf.d/
   ```

6. Write the BOS config into `/usr/local/etc/sc-bos/`: `system.conf.json` with the settings from
   `system.conf.json.example`, `app.conf.json`, and the Ops UI config in `ui-config/`. Make the folders `0750` and
   the files `0640`, owned by `root:sc-bos`. Check the [listening ports](#listening-ports).

7. Build a release of the version to run with `scripts/build-release.sh`, and copy the tarball to the host. For
   example, from this repo:

   ```sh
   bash scripts/build-release.sh --version v2026.10.1 --arch amd64
   ```

   Use `--arch arm64` for an arm64 host. See [Building an artefact](release-artefact.md#building-an-artefact) for
   the options, including building a product's own main package.

8. Enable and start the Supervisor:

   ```sh
   sysrc sc_bos_supervisor_enable=YES
   service sc_bos_supervisor start
   ```

9. Install the release:

   ```sh
   sc-bos-supervisor install-release sc-bos-v2026.10.1-freebsd-amd64.tar.gz
   ```

   It prints `installed v2026.10.1; current -> v2026.10.1 (was: none)`.

10. Enable and start BOS:

    ```sh
    sysrc sc_bos_enable=YES
    service sc_bos start
    ```

11. Enrol the node from the Ops UI (optional).

### The install-release command

```
sc-bos-supervisor install-release [-config FILE] [-restart] [-lock-timeout DUR] TARBALL
```

`install-release` installs a release from a local tarball into the releases root, and points `current` at it. It
reads the version from the tarball's manifest. Run it as root. Put `install-release` before any flags.

| Flag                | Meaning                                                                          |
|---------------------|----------------------------------------------------------------------------------|
| `-config FILE`      | Supervisor config file. Default `/usr/local/etc/sc-bos-supervisor/config.json`.  |
| `-restart`          | Run `service sc_bos restart` after installing. Without it, nothing is restarted. |
| `-lock-timeout DUR` | How long to wait if the Supervisor is installing an update. Default `30s`.       |

It can run whether the Supervisor is running or not. Locks are used to prevent conflicts.

Moving an area controller under the Supervisor
----------------------------------------------

Existing area controllers run BOS as the `areacontroller` rc service, set up by an Ansible `areacontroller` role:

| What             | Where                                                                                         |
|------------------|-----------------------------------------------------------------------------------------------|
| Service and user | `areacontroller` rc service, running as the `areacontroller` user and group under `daemon -r` |
| rc script        | `/usr/local/opt/areacontroller/etc/rc.d/areacontroller`, linked from `/usr/local/etc/rc.d/`   |
| Install folder   | `/usr/local/opt/areacontroller`, `root:areacontroller` `0755`                                 |
| Binary           | `bin/area-controller`                                                                         |
| Config           | `conf/system.json` and `conf/area-controller.local.json`                                      |
| Data             | `data/`, owned by `areacontroller`                                                            |
| Ops UI           | `static/ops`, served by the `staticHosting` entry for `/`                                     |
| Log              | `/var/log/areacontroller.log`                                                                 |

The move keeps the service, user, config, data and log, and BOS already serves HTTPS on port 8443. Only the binary
and the Ops UI move, into a releases root at `/usr/local/opt/areacontroller/releases`. Existing data, including any
cloud registration, carries over.

1. Build a release in the Go module that builds the area controllers' BOS binary, from the main package they run.
   The module must use an sc-bos version that supports the Supervisor integration.

   ```sh
   go mod download github.com/smart-core-os/sc-bos
   bash "$(go list -m -f '{{.Dir}}' github.com/smart-core-os/sc-bos)/scripts/build-release.sh" --main ./cmd/area-controller
   ```

2. Install the Supervisor, as in steps 3, 4 and 8 of the [fresh install](#fresh-install): its binary, rc script and
   newsyslog config. Don't install the `sc_bos` rc script. Use this Supervisor config instead of the reference one:

   ```json
   {
     "installer": "freebsdrc",
     "socketGroup": "areacontroller",
     "freebsdrc": {
       "root": "/usr/local/opt/areacontroller/releases",
       "service": "areacontroller"
     }
   }
   ```

3. Copy the tarball to the host and install it. This creates the releases root and points `current` at the release:

   ```sh
   sc-bos-supervisor install-release sc-bos-<version>-freebsd-amd64.tar.gz
   ```

4. Change the Ansible `areacontroller` role and config, then deploy them:
   - In the rc script, `areacontroller-rc.sh`, run `/usr/local/opt/areacontroller/releases/current/bin/sc-bos`
     instead of `bin/area-controller`, and add `sc_bos_supervisor` to the `REQUIRE` line.
   - In `system.json`, add `"supervisor": {"enabled": true}`, and change the `staticHosting` entry for `/` to
     `/usr/local/opt/areacontroller/releases/current/ui/ops`.
   - Remove the tasks that build and copy `bin/area-controller`, and stop any deployment of the Ops UI to
     `static/ops`. Otherwise Ansible undoes Supervisor updates.
5. Restart BOS with `service areacontroller restart`, then check that it committed: the Supervisor log shows
   `committed version changed` with the release version, and `/var/db/sc-bos-supervisor/state.json` shows it as
   `committed`.

Keep `bin/area-controller` and `static/ops` until the first Supervisor update has completed. To undo the move,
revert the Ansible changes and deploy them again.

Logs and troubleshooting
------------------------

| Log                              | Written by                                         |
|----------------------------------|----------------------------------------------------|
| `/var/log/sc-bos-supervisor.log` | the Supervisor, as JSON, apart from startup errors |
| `/var/log/sc-bos.log`            | BOS                                                |

`newsyslog` rotates both at 10 MB and keeps 7 old files.

Each update ends with one Supervisor log line. If the Supervisor stops during an update, the line comes once it
resumes the update on its next start.

- **`update completed`**, at `info`, with the new `version`.
- **`update failed`**, at `warn`, with the `version` and a `reason`, such as `download: ...`, `apply: ...` or
  `update v2026.11.0 not confirmed within 5m0s`.

Along the way, a successful install logs `switched current release` with the new `current` and the old one as
`previous`. An update that BOS doesn't commit in time logs `update not confirmed, rolling back` before it fails, and
`current` points back at the previous release.

Until its first update finishes, the Supervisor logs `committed version changed` when BOS commits a different
`version` from the one it last recorded, with the old one as `previous`. After an update finishes, it keeps that
update's committed version until an update completes, and doesn't log a different commit.

The Supervisor also keeps its update state in `/var/db/sc-bos-supervisor/state.json`. To check it, and the releases
on disk:

```sh
cat /var/db/sc-bos-supervisor/state.json
ls -l /usr/local/sc-bos/releases
```

Common problems:

- **The Supervisor log repeats `fatal: invalid config ...: socketGroup: group: unknown group sc-bos`.** `service
  sc_bos_supervisor start` still reports success, but `daemon` restarts the failing Supervisor every second. Create
  the `sc-bos` group, or set `socketGroup` to the group BOS runs as. The same pattern, a `fatal:` line repeated
  every second, applies to any other startup error.
- **Every update rolls back.** BOS isn't committing. Check that `sc_bos_enable` is `YES`: if it isn't,
  `service sc_bos restart` doesn't restart BOS, so the new release never runs. Check the BOS log for
  `supervisor commit failed, will retry`.
  A permission error means BOS's group can't use the socket: check `socketGroup` and
  `ls -l /var/run/sc-bos-supervisor`. If BOS is enrolled, it also waits for a successful cloud check-in before it
  commits. A release built without a version can't commit either; build releases with `scripts/build-release.sh`.
- **The Supervisor logs `update failed` with a `reason` starting `apply: restart sc_bos:`.** `service sc_bos
  restart` failed or took longer than 5 minutes. Run `service sc_bos restart` by hand to see the error.
- **BOS doesn't start, and its log shows `permission denied` on a listen address.** See
  [listening ports](#listening-ports).
- **`install-release` fails with `the Supervisor has an unfinished update`, or `an update is in progress`.** Wait
  for the update to finish, then run it again.
