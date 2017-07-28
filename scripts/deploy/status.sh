#!/bin/bash

KERNEL_LOCAL="3.10.0-327.37.el7.lvs.x86_64"

now_version=`uname -r`
if [ x"${now_version}" != x"$KERNEL_LOCAL" ];then
    echo $now_version
    echo "kernel update fail"
    exit 1
fi

function check_status() {
    name=$1
    cnt=1
    while [ $cnt -le 3 ];
    do
        systemctl status $name
        if [ $? -ne 0 ];then
            echo "${name} is not started"
            let cnt=$cnt+1
            sleep 5
        else
            return 0
        fi
    done
    exit 1
}
# flannel status
# check_status flanneld

# lvs status
check_status lvs

# zebra status
check_status zebra

# ospfd status
check_status ospfd

# keepalived status
check_status keepalived

# lbmanager status
check_status lbmanager

ip link show br-lvs
if [ $? -ne 0 ];then
    echo "br-lvs not exist"
    exit 1
fi

exit 0
