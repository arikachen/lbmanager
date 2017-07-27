#!/bin/bash
# if you env behind proxy, pls set it first

KERNER_VER=$1

RPM_SOURCE_DIR=/root/rpmbuild/SOURCES
RPM_SPEC_DIR=/root/rpmbuild/SPECS
RPM_SPEC=$RPM_SPEC_DIR/kernel.spec

RPM_TMP_DIR=/tmp/build
KERNEL_RPM=kernel-3.10.0-327.el7.src.rpm


PATCH1=0001-ali-lvs-for-centos7.2.patch
PATCH2=0002-lvs-add-lvs-cqs-support.patch
PATCH3=0003-lvs-add-cps-and-bps-limit-support.patch
PATCH4=0004-lvs-fix-add-no-toa-when-tcp-header-full.patch

mkdir -p $RPM_TMP_DIR
# 1. download and install the kernel source
wget -T 60 http://vault.centos.org/7.2.1511/os/Source/SPackages/$KERNEL_RPM -O $RPM_TMP_DIR/$KERNEL_RPM
if [ $? -ne 0 ];then
    echo "download kernel source fail"
    exit 1
fi

groupadd mockbuild
useradd -g mockbuild mockbuild

rpm -ivh $RPM_TMP_DIR/$KERNEL_RPM --force
rm -rf $RPM_TMP_DIR

# 2. install the dependence
yum install -y openssl-devel krb5-devel ncurses-devel rng-tools rpm-build bc patch xmlto asciidoc  hmaccalc python-devel  newt-devel pesign elfutils-devel binutils-devel bison audit-libs-devel numactl-devel pciutils-devel perl-ExtUtils-Embed
service rngd restart

# 3. modify the source
cp kernel/* $RPM_SOURCE_DIR

if [ ! -f $RPM_SPEC ];then
    echo "kernel spec not found"
    exit 1
fi

grep $PATCH1 $RPM_SPEC
if [ $? -ne 0 ];then
    sed -i "s/specrelease 327/specrelease ${KERNER_VER}/" $RPM_SPEC
    li=`grep -n Patch1002 ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`

    insert="${li} aPatch1003: ${PATCH1}\nPatch1004: ${PATCH2}\nPatch1005: ${PATCH3}\nPatch1006: ${PATCH4}"
    sed -i "${insert}" $RPM_SPEC

    li=`grep -n "ApplyOptionalPatch debrand-rh-i686-cpu.patch" ${RPM_SPEC} |head -1 | cut  -d  ":"  -f  1`
    insert="${li} aApplyOptionalPatch ${PATCH1}\nApplyOptionalPatch ${PATCH2}\nApplyOptionalPatch ${PATCH3}\nApplyOptionalPatch ${PATCH4}"
    sed -i "${insert}" $RPM_SPEC
fi

KERNEL_CONF=$RPM_SOURCE_DIR/kernel-3.10.0-x86_64.config
sed -i "s/^CONFIG_NETFILTER_XT_MATCH_IPVS.*/\# CONFIG_NETFILTER_XT_MATCH_IPVS is not set/" $KERNEL_CONF
sed -i "s/^CONFIG_IP_VS_TAB_BITS=.*/CONFIG_IP_VS_TAB_BITS=22/" $KERNEL_CONF

if [ -f /etc/rpm/macros.dist ];then
    sed -i "s/%dist.*/%dist .el7.lvs/g" /etc/rpm/macros.dist
fi

# 4. build rpm
rpmbuild -bb --with baseonly --with firmware --without kabichk --without debuginfo --target=`uname -m` $RPM_SPEC
if [ $? -ne 0 ];then
    echo "kernel rpmbuild failed"
    exit 1
fi

exit 0
