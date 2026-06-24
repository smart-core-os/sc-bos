# Apple Container machine image for the supervisor update demo.
FROM rockylinux/rockylinux:10

# Set up machine with systemd, so it can run as a VM
RUN dnf -y install \
        systemd dbus podman golang \
        git jq curl-minimal tar gzip sudo \
        procps-ng iproute hostname ca-certificates \
    && dnf clean all \
    && rm -rf /var/cache/dnf

# Passwordless sudo for the mapped user (container machine adds it to wheel when it provisions the account)
RUN echo '%wheel ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/wheel-nopasswd \
    && chmod 0440 /etc/sudoers.d/wheel-nopasswd

# systemd-in-a-container housekeeping: clear the seeded machine-id (regenerated on first boot) and
# mask units that have nothing to do in a VM.
RUN : > /etc/machine-id
RUN systemctl set-default multi-user.target \
    && systemctl mask \
        dev-hugepages.mount \
        sys-fs-fuse-connections.mount \
        systemd-update-utmp.service

# Boot systemd so `systemctl` (and the demo's Quadlet units) work inside the machine.
CMD ["/sbin/init"]
