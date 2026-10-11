Name:           sfos_airpods
Version:        0.1.0
Release:        1
Summary:        AirPods controls for Sailfish OS
License:        GPLv3
URL:            https://github.com/mrcyjanek/sfos_airpods
Source0:        %{name}-%{version}.tar.bz2
Requires:       %{name}-daemon = %{version}-%{release}
Requires:       sailfishsilica-qt5
BuildRequires:  systemd
BuildRequires:  pkgconfig(sailfishapp)
BuildRequires:  pkgconfig(Qt5Core)
BuildRequires:  pkgconfig(Qt5Gui)
BuildRequires:  pkgconfig(Qt5Qml)
BuildRequires:  pkgconfig(Qt5Quick)
BuildRequires:  desktop-file-utils

%description
Silica page for AirPods battery, noise control, and ear detection.

%package daemon
Summary:        AirPods control daemon
Provides:       sfos-airpods-daemon = %{version}-%{release}
Requires:       systemd
Requires(post): systemd
Requires(preun): systemd

%description daemon
System daemon that owns the AirPods control channel.

%prep
%setup -q

%build
%qmake5
%make_build
# Go itself is installed by .sfosbuild/10-golang.sh before this runs.
CGO_ENABLED=0 go build -trimpath -o airpodsd ./cmd/airpodsd
CGO_ENABLED=0 go build -trimpath -o airpods ./cmd/airpods

%install
%qmake5_install
desktop-file-install --delete-original \
    --dir %{buildroot}%{_datadir}/applications \
    %{buildroot}%{_datadir}/applications/*.desktop
install -D -m 0755 airpodsd %{buildroot}%{_bindir}/airpodsd
install -D -m 0755 airpods %{buildroot}%{_bindir}/airpods
install -D -m 0644 systemd/sfos_airpods.service %{buildroot}/usr/lib/systemd/system/sfos_airpods.service
install -D -m 0644 dbus/net.mrcyjanek.sfos_airpods.conf %{buildroot}%{_datadir}/dbus-1/system.d/net.mrcyjanek.sfos_airpods.conf
install -D -m 0644 sysusers.d/sfos_airpods.conf %{buildroot}/usr/lib/sysusers.d/sfos_airpods.conf

%post daemon
systemd-sysusers /usr/lib/sysusers.d/sfos_airpods.conf >/dev/null 2>&1 || \
    useradd -r -g bluetooth -s /usr/sbin/nologin -c "AirPods control" airpods >/dev/null 2>&1 || :
systemctl daemon-reload >/dev/null 2>&1 || :
systemctl enable sfos_airpods.service >/dev/null 2>&1 || :
systemctl restart sfos_airpods.service >/dev/null 2>&1 || :

%preun daemon
if [ "$1" -eq 0 ]; then
    systemctl disable --now sfos_airpods.service >/dev/null 2>&1 || :
fi

%postun daemon
systemctl daemon-reload >/dev/null 2>&1 || :

%files
%{_bindir}/sfos_airpods
%{_datadir}/applications/sfos_airpods.desktop
%{_datadir}/icons/hicolor/*/apps/%{name}.png
%{_datadir}/sfos_airpods

%files daemon
%{_bindir}/airpodsd
%{_bindir}/airpods
/usr/lib/systemd/system/sfos_airpods.service
%{_datadir}/dbus-1/system.d/net.mrcyjanek.sfos_airpods.conf
/usr/lib/sysusers.d/sfos_airpods.conf
