Name:           sc-bos-supervisor
Version:        %{?version}%{!?version:0.0.0}
Release:        1%{?dist}
Summary:        Smart Core BOS update Supervisor

License:        GPL-3.0-or-later
URL:            https://github.com/smart-core-os/sc-bos

BuildRequires:  systemd-rpm-macros
%{?systemd_requires}

# The binary is built ahead of time by supervisor/rpm/build.sh and staged into %{_sourcedir}; this
# spec only packages it. Building the whole monorepo (UI + deps) inside an rpmbuild sandbox would be
# far heavier for no benefit, and the version is injected via -ldflags at that build step.
Source0:        sc-bos-supervisor
Source1:        sc-bos-supervisor.service
Source2:        config.json

# A near-static CGO_ENABLED=0 Go binary: no shared-library autodeps to find.
%global debug_package %{nil}
%global __requires_exclude .*

%description
The Smart Core BOS Supervisor: a privileged system service that installs BOS software updates
out-of-process, rolls them back locally if the new version is unhealthy, and updates itself.

%prep
# nothing to do: sources are prebuilt artefacts, not a source tarball.

%build
# nothing to do: the binary is prebuilt (see supervisor/rpm/build.sh).

%install
install -Dm 0755 %{SOURCE0} %{buildroot}%{_bindir}/sc-bos-supervisor
install -Dm 0644 %{SOURCE1} %{buildroot}%{_unitdir}/sc-bos-supervisor.service
install -Dm 0644 %{SOURCE2} %{buildroot}%{_sysconfdir}/sc-bos-supervisor/config.json

%post
%systemd_post sc-bos-supervisor.service

%preun
%systemd_preun sc-bos-supervisor.service

%postun
# On upgrade this restarts the running Supervisor onto the newly installed binary. The self-update
# applier (running outside this unit's cgroup) relies on that restart to bring the new version up.
%systemd_postun_with_restart sc-bos-supervisor.service

%files
%{_bindir}/sc-bos-supervisor
%{_unitdir}/sc-bos-supervisor.service
%dir %{_sysconfdir}/sc-bos-supervisor
%config(noreplace) %{_sysconfdir}/sc-bos-supervisor/config.json

%changelog
* Mon Jun 23 2025 Smart Core <noreply@smart-core-os.org> - 0.0.0-1
- Package the BOS update Supervisor.
