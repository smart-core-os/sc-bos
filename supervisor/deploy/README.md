BOS Supervisor — deployment units (Rocky Linux / Podman)
========================================================

Two units make up a Supervisor-managed BOS deployment. See
[`cmd/supervisor/docs/supervisor-mechanism-rocky.md`](../../cmd/supervisor/docs/supervisor-mechanism-rocky.md)
for the mechanism.

Install
-------
The Supervisor ships as the `sc-bos-supervisor` RPM (built by `supervisor/rpm/build.sh`), which
installs the binary to `/usr/bin/sc-bos-supervisor`, this unit to `%{_unitdir}`, and the default
config to `/etc/sc-bos-supervisor/config.json`. Installing the RPM also enables the service via the
`%post` scriptlet; a later upgrade replaces the binary in place and restarts the unit (the basis for
Supervisor self-update).

   ```sh
   dnf install ./sc-bos-supervisor-*.rpm
   cp sc-bos.container /etc/containers/systemd/   # Quadlet generates sc-bos.service (not in the RPM)
   systemctl daemon-reload
   systemctl enable --now sc-bos-supervisor.service
   systemctl start sc-bos.service
   ```

`sc-bos.container`
------------------
Quadlet unit for BOS. The Supervisor depends only on `Image=…:current` (the tag it swaps); `HealthCmd`
is kept for systemd's own `Restart=` and observability, not as the success/rollback gate (BOS confirms an
update by calling the Commit RPC). It mounts the Supervisor socket *directory* `/run/sc-bos-supervisor`
so BOS can reach the API at `/run/sc-bos-supervisor/supervisor.sock`.

`sc-bos-supervisor.service`
---------------------------
Runs the Supervisor binary as root. `ExecStart` points at `/usr/bin/sc-bos-supervisor`, the
RPM-managed path, so a Supervisor self-update is an `rpm`/`dnf` upgrade that replaces the file in
place; the package's `%systemd_postun_with_restart` restarts this unit onto the new binary.

`config.json`
-------------
Optional. The Supervisor reads `/etc/sc-bos-supervisor/config.json` (override with `-config`); if absent
it uses built-in defaults, and any option the file omits keeps its default. Options: `socket`, `stateDir`
(holds `state.json` and `staging/`), `imageRepo`, `unit`, and `commitDeadline` (a Go duration, e.g.
`"2m"`). The sample here lists the defaults.

SELinux
-------
Rocky enforces SELinux by default. The socket directory mount uses `:z` to relabel it, matching the
data volumes. Beyond labelling, a confined BOS container domain writing to a socket created by the
Supervisor domain may still be denied by type enforcement — validate on an enforcing host (see the
plan's verification checklist), capture any AVC denial, and apply the minimal policy/context fix.
