#!/bin/bash

KEEPALIVED_CONF=/etc/keepalived/keepalived.conf
SERVICE_CONF=/etc/keepalived/conf.d
SERVICE_COMMON_CONF=/etc/keepalived/common.conf.d

LOCAL_IP_RANGE=$1
SECOND_IP_RANGE=$2

cat > $KEEPALIVED_CONF <<EOF
! Configuration File for keepalived

global_defs {
#   notification_email {
#     acassen@firewall.loc
#     failover@firewall.loc
#     sysadmin@firewall.loc
#   }
#   notification_email_from Alexandre.Cassen@firewall.loc
#   smtp_server 192.168.200.1
#   smtp_connect_timeout 30
#   router_id LVS_DEVEL
}

local_address_group laddr_g1 {
   $LOCAL_IP_RANGE
}

local_address_group laddr_g2 {
   $SECOND_IP_RANGE
}

EOF

mkdir -p $SERVICE_CONF
echo "include conf.d/*.conf" >> $KEEPALIVED_CONF
mkdir -p $SERVICE_COMMON_CONF
echo "include common.conf.d/*.conf" >> $KEEPALIVED_CONF
