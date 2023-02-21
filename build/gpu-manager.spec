Name: gpu-manager
Version: %{version}
Release: %{commit}%{?dist}
Summary: GPU Manager Plugin for Kubernetes

License: MIT
Source: gpu-manager-source.tar.gz

Requires: systemd-units

%define pkgname %{name}-%{version}-%{release}

%description
GPU Manager Plugin for Kubernetes

%prep
%setup -n gpu-manager-%{version}


%build
make all

#https://blog.csdn.net/hncomputer/article/details/7049127
#https://www.cnblogs.com/schangech/p/5641108.html

%install
install -d $RPM_BUILD_ROOT/%{_bindir}
install -d $RPM_BUILD_ROOT/%{_unitdir}
install -d $RPM_BUILD_ROOT/etc/gpu-manager
#-d: 创建目录并进入该目录

install -p -m 755 ./go/bin/gpu-manager $RPM_BUILD_ROOT/%{_bindir}/
install -p -m 755 ./go/bin/gpu-client $RPM_BUILD_ROOT/%{_bindir}/
#-p:以<来源>文件的访问/修改时间作为相应的目的地文件的时间属性
#-m:复制文件并设置权限模式

install -p -m 644 ./build/extra-config.json $RPM_BUILD_ROOT/etc/gpu-manager/
install -p -m 644 ./build/gpu-manager.conf $RPM_BUILD_ROOT/etc/gpu-manager/
install -p -m 644 ./build/volume.conf $RPM_BUILD_ROOT/etc/gpu-manager/

install -p -m 644 ./build/gpu-manager.service $RPM_BUILD_ROOT/%{_unitdir}/

%clean
rm -rf $RPM_BUILD_ROOT

%files
%config(noreplace,missingok) /etc/gpu-manager/extra-config.json
%config(noreplace,missingok) /etc/gpu-manager/gpu-manager.conf
%config(noreplace,missingok) /etc/gpu-manager/volume.conf
#配置文件不被修改

/%{_bindir}/gpu-manager
/%{_bindir}/gpu-client

/%{_unitdir}/gpu-manager.service
