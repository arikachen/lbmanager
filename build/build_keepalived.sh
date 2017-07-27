#!/bin/bash

RPM_SOURCE_DIR=/root/rpmbuild/SOURCES
RPM_SPEC_DIR=/root/rpmbuild/SPECS
RPM_SPEC=$RPM_SPEC_DIR/keepalived.spec
KEEPALIVED_SUB_VERSION=12
RPM_TMP_DIR=/tmp/build

echo '%debug_package %{nil}' >> ~/.rpmmacros

KEEPALIED_RPM=keepalived-1.2.13-7.el7.src.rpm
PATCH1=0001-lvs-add-fullnat-and-synproxy-support.patch
PATCH2=0002-lvs-add-bps-and-cps-limit-in-keepalived.patch
PATCH3=0003-lvs-use-epoll-optimize-keepalived.patch
PATCH4=0004-lvs-fix-rs-not-active-sometime-when-host-reboot.patch
PATCH5=0005-lvs-tcp-check-supports-retry.patch

mkdir -p $RPM_TMP_DIR
# 1. download and install the kernel source
wget -T 60 http://vault.centos.org/7.2.1511/os/Source/SPackages/$KEEPALIED_RPM -O $RPM_TMP_DIR/$KEEPALIED_RPM
if [ $? -ne 0 ];then
    echo "download keepalived source fail"
    exit 1
fi

groupadd mockbuild
useradd -g mockbuild mockbuild

rpm -ivh $RPM_TMP_DIR/$KEEPALIED_RPM --force

if [ ! -f $RPM_SPEC ];then
    echo "keepalived spec not found"
    exit 1
fi

rm -rf $RPM_TMP_DIR

# 2. install the dependence
yum install -y rpm-build gcc openssl-devel libnl-devel
rpm -e libnl3-devel

# 3. modify the source
cp keepalived/* $RPM_SOURCE_DIR

grep $PATCH1 $RPM_SPEC
if [ $? -ne 0 ];then
    li=`grep -n Patch1 ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`

    insert="${li} aPatch2: ${PATCH1}\nPatch3: ${PATCH2}\nPatch4: ${PATCH3}\nPatch5: ${PATCH4}\nPatch6: ${PATCH5}"
    sed -i "${insert}" $RPM_SPEC

    li=`grep -n "patch1 " ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`
    insert="${li} a%patch2 -p1\n%patch3 -p1\n%patch4 -p1\n%patch5 -p1\n%patch6 -p1"
    sed -i "${insert}" $RPM_SPEC
    sed -i "s/libnl3-devel/libnl-devel/g" $RPM_SPEC
    # change release version
    sed -i "s/Release: 7/Release: ${KEEPALIVED_SUB_VERSION}/g" $RPM_SPEC

    # add libipvs package
    li=`grep -n "VRRPv2" ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`
    li=`expr $li + 1`
    insert="${li} a%package -n libipvs\\nSummary: libipvs\\n%description -n libipvs\\nipvs lib for ipvsadm\\n"
    sed -i "${insert}" $RPM_SPEC

    li=`grep -n "keepalived.8" ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`
    li=`expr $li + 1`
    insert="${li} a%files -n libipvs\\n%{_libdir}/lib*.a\\n%{_includedir}/*\\n"
    sed -i "${insert}" $RPM_SPEC
fi

if [ -f /etc/rpm/macros.dist ];then
    sed -i "s/%dist.*/%dist .el7.lvs/g" /etc/rpm/macros.dist
fi

rpmbuild -bb --without snmp $RPM_SPEC
if [ $? -ne 0 ];then
    echo "keepalived rpmbuild failed"
    exit 1
fi

exit 0

