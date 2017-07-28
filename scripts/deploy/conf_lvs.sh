#!/bin/bash

ROOT=$(cd "$(dirname "$0")"; pwd)

# format 192.168.1.1 or 192.168.1.1-5
# not support 192.168.1.5-1
LOCAL_IP_MASK=`echo ${DOWN_LINK_IP} | awk -F"/" '{print $2}'`

# rfc
rfc=4096
cc=$(grep -c processor /proc/cpuinfo)
rsfe=$(echo $cc*$rfc | bc)

function set_rfc() {
    nic=$1
    for file_rfc in $(ls /sys/class/net/${nic}/queues/rx-*/rps_flow_cnt)
    do
        echo $rfc > $file_rfc
    done
}

SYSCTL=/etc/sysctl.conf
function modify_syscfg(){
    key=$1
    val=$2
    grep $key $SYSCTL
    if [ $? -ne 0 ]; then
        echo "${key} = ${val}" >> $SYSCTL
    else
        repl="s/${key}.*/${key} = ${val}/g"
        sed -i "${repl}" $SYSCTL
    fi
}
# 1. set syscfg
sed -i "/nf_conntrack*/d" /etc/sysctl.conf

modify_syscfg net.core.netdev_max_backlog 500000
modify_syscfg net.ipv4.conf.all.arp_ignore 1
modify_syscfg net.ipv4.conf.all.arp_announce 2
modify_syscfg net.core.rps_sock_flow_entries $rsfe
sysctl -p
if [ $? -ne 0 ];then
    echo "modify syscfg fail"
    exit 1
fi
sleep 1

# 2. close irqbalance
if [ -f /usr/lib/systemd/system/irqbalance.service ];then
    systemctl stop irqbalance > /dev/null
    systemctl disable irqbalance
fi

# 3. set irq
ethtool -K $UP_LINK gro off
ethtool -K $UP_LINK lro off
dv=`ethtool -i $UP_LINK |grep driver |awk '{print $2}'`
if [ x${dv} == x"bonding" ];then
    bond=/proc/net/bonding/${UP_LINK}
    if [ -f ${bond} ];then
        bl=`grep "Slave Interface" $bond |awk '{print $3}'`
        for eth in $bl
        do
            ethtool -K $eth gro off
            ethtool -K $eth lro off
            $ROOT/nic.sh -i $eth
            set_rfc $eth
        done
    fi
else
    $ROOT/nic.sh -i $UP_LINK
    set_rfc $UP_LINK
fi

# 4. config local ip todo
ip addr show $DOWN_LINK
if [ $? -ne 0 ];then
    brctl addbr $DOWN_LINK
    if [ $? -ne 0 ];then
        echo "add bridge failed"
        exit 1
    fi
fi

ip addr show $DOWN_LINK |grep $DOWN_LINK_IP
if [ $? -ne 0 ];then
    ifconfig $DOWN_LINK $DOWN_LINK_IP mtu $DOWN_LINK_MTU up
    if [ $? -ne 0 ];then
        echo "set bridge ip failed"
        exit 1
    fi
fi

function set_local_ip()
{
    ip_range=$1
    ip_mask=$2
    dev=$3
    IFS_old=$IFS
    IFS='-' arr=($ip_range)
    end=${arr[1]}
    IFS='.' ip_arr=(${arr[0]})
    IFS=$IFS_old
    if [ ${#ip_arr[@]} -ne 4 ];then
        echo "wrong ip format"
        exit 1
    fi
    if [ -z $end ];then
        ip addr add ${arr[0]}/$ip_mask dev $dev
        exit 0
    fi

    start=${ip_arr[3]}
    while [ $start -le $end ];
    do
        lip=${ip_arr[0]}.${ip_arr[1]}.${ip_arr[2]}.$start
        ip addr add $lip/$ip_mask dev $dev
        start=`expr $start + 1`
    done
}

set_local_ip $LOCAL_IP_RANGE $LOCAL_IP_MASK $DOWN_LINK

UP_LINK_IP=`ifconfig ${UP_LINK} |grep "inet " |awk '{print $2}'`
if [ x"${UP_LINK_IP}" != x"$SECOND_IP_RANGE" ];then
    set_local_ip $SECOND_IP_RANGE $SECOND_IP_MASK $DOWN_LINK
fi

exit 0
