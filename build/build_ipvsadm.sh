#!/bin/bash

RPM_SOURCE_DIR=/root/rpmbuild/SOURCES

echo '%debug_package %{nil}' >> ~/.rpmmacros
if [ -f /etc/rpm/macros.dist ];then
    sed -i "s/%dist.*/%dist .el7.lvs/g" /etc/rpm/macros.dist
fi

cp ipvsadm/* $RPM_SOURCE_DIR
rpmbuild -bb ipvsadm/ipvsadm.spec
if [ $? -ne 0 ];then
    echo "ipvsadm rpmbuild failed"
    exit 1
fi

exit 0

