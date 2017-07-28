#!/bin/bash

QUAGGA_CONF_PATH=/etc/quagga
ZEBRA_CONF=$QUAGGA_CONF_PATH/zebra.conf
OSPFD_CONF=$QUAGGA_CONF_PATH/ospfd.conf
PASSWD=123456

OSPF_DIGEST_KEY=8
OSPF_HELLO_TIME=3
OSPF_DEAD_TIME=12
OSPF_COST=10

if [ $# -lt 4 ];then
    echo "usage: bash conf_quagga.sh [nic] [area] [vip_cidr] [route_filter]"
    exit 1
fi

NIC=$1
OSPF_AREA=$2
OSPF_VIP=$3
OSPF_ROUTE_FILTER=$4

OSPF_RID=`ifconfig ${NIC} |grep "inet " |awk '{print $2}'`
if [ -z ${OSPF_RID} ];then
    echo "get ospf router id fail"
    exit 1
fi

OSPF_INTERNAL=`ip route show |grep "${OSPF_RID}" |grep ${NIC} |awk '{print $1}'`
if [ -z ${OSPF_INTERNAL} ];then
    echo "get ospf internal cidr fail"
    exit 1
fi


# 1 install quagga
yum install -y quagga


hostname=`uname -n`

cat > $ZEBRA_CONF <<EOF
hostname zebra.${hostname}
password 8 ${PASSWD}
enable password 8 ${PASSWD}
log file /var/log/quagga/zebra.log
service password-encryption
EOF

cat > $OSPFD_CONF <<EOF
hostname ospfd.${hostname}
password 8 ${PASSWD}
enable password 8 ${PASSWD}
log file /var/log/quagga/ospf.log
service password-encryption

interface ${NIC}
  #ip ospf message-digest-key ${OSPF_DIGEST_KEY} md5 ${PASSWD}
  ip ospf hello-interval ${OSPF_HELLO_TIME}
  ip ospf dead-interval ${OSPF_DEAD_TIME}
  ip ospf cost ${OSPF_COST}
router ospf
  ospf router-id ${OSPF_RID}
  log-adjacency-changes
  network ${OSPF_INTERNAL} area ${OSPF_AREA}
  #area ${OSPF_AREA} authentication message-digest
  #area ${OSPF_AREA} stub no-summary
  network ${OSPF_VIP} area ${OSPF_AREA}
EOF

# ospf route filter
IFS_old=$IFS
IFS=',' route_arr=(${OSPF_ROUTE_FILTER})
IFS=$IFS_old
arr_len=${#route_arr[@]}
if [ ${arr_len} -eq 0 ];then
    echo "wrong number of route filter"
    exit 1
fi

cnt=0
route_filter="\n  area ${OSPF_AREA} filter-list prefix LVS-VIP in\n  !\n"
while [ ${cnt} -lt ${arr_len} ];
do
    let seq=($cnt+1)*10
    route_filter+="  ip prefix-list LVS-VIP seq ${seq} permit ${route_arr[$cnt]} le 32\n"
    let cnt=$cnt+1
done
route_filter+="  ip prefix-list LVS-VIP seq 100 deny 0.0.0.0/0 le 32"
#echo -e "${route_filter}" >> $OSPFD_CONF

systemctl enable zebra
systemctl enable ospfd

systemctl restart zebra
if [ $? -ne 0 ];then
    echo "start zebra failed"
    exit 1
fi

systemctl restart ospfd
if [ $? -ne 0 ];then
    echo "start ospfd failed"
    exit 1
fi

echo "config quagga end"
exit 0
