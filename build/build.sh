#!/bin/bash

RPM_ROOT=/root/rpmbuild/RPMS/x86_64
KERNER_SUB_VER=327.37
KERNEL_VER=3.10.0-$KERNER_SUB_VER

echo "Build kernel keepalived and ipvsadm..."

# 1. build kernel
sh build_centos7.2.sh $KERNER_SUB_VER
if [ $? -ne 0 ];then
    echo "build kernel failed"
    exit 1
fi

function clean_rpm(){
    rpm -e kernel-${KERNEL_VER}* &> /dev/null
    rpm -e kernel-devel-${KERNEL_VER}* &> /dev/null
    rpm -e kernel-headers-${KERNEL_VER}* &> /dev/null
    rpm -e kernel-tools-${KERNEL_VER}* &> /dev/null
    rpm -e kernel-tools-libs-${KERNEL_VER}* &> /dev/null
    rpm -e libipvs* &> /dev/null
    yum reinstall -y kernel-headers &> /dev/null
}

# 2. build keepalived
# kernel header need by keepalived
rpm -ivh $RPM_ROOT/kernel-${KERNEL_VER}*.rpm --force
rpm -ivh $RPM_ROOT/kernel-devel-${KERNEL_VER}*.rpm --force
rpm -ivh $RPM_ROOT/kernel-headers-${KERNEL_VER}*.rpm --force
rpm -ivh $RPM_ROOT/kernel-tools-libs-${KERNEL_VER}*.rpm --force
rpm -ivh $RPM_ROOT/kernel-tools-${KERNEL_VER}*.rpm --force

sh build_keepalived.sh
if [ $? -ne 0 ];then
    echo "build keepalived failed"
    clean_rpm
    exit 1
fi

# 3. build ipvsadm
rpm -ivh $RPM_ROOT/libipvs*.rpm --force
sh build_ipvsadm.sh
if [ $? -ne 0 ];then
    echo "build ipvsadm failed"
    clean_rpm
    exit 1
fi

clean_rpm

# 4. uplaod binary
echo  "TODO uplaod binary"

exit 0
