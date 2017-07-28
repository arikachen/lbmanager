#!/bin/bash

ROOT=$(cd "$(dirname "$0")"; pwd)
GRUB_CONF=/etc/default/grub
LVS_ROOM=/opt/lvs
DOWN_LINK=br-lvs

RPM_ROOT=/root/rpmbuild/RPMS/x86_64

if [ $# -lt 1 ];then
    echo "usage: bash deploy.sh [envfile]"
    exit 1
fi

ENV_FILE=$1
if [ ! -f $ENV_FILE ];then
    echo "env file is not exist"
    exit 1
fi

source $ENV_FILE

cd $ROOT
touch *

ip link show $UP_LINK
if [ $? -ne 0 ];then
    echo "uplink not exist"
    exit 1
fi

UP_LINK_IP=`ifconfig ${UP_LINK} |grep "inet " |awk '{print $2}'`

systemctl status flanneld
if [ $? -ne 0 ];then
    echo "flannel is not started"
    exit 1
fi

if [ ! -f /run/flannel/subnet.env ];then
    echo "flannel subnet not found"
    exit 1
fi

source /run/flannel/subnet.env

function stop_service()
{
    name=$1
    if [ -f /usr/lib/systemd/system/$name.service ];then
        systemctl stop $name > /dev/null
    fi
}

function disable_conntrack()
{
cat > /etc/modprobe.d/blacklist_lvs.conf <<EOF
blacklist nf_conntrack
blacklist nf_conntrack_ipv6
blacklist xt_conntrack
blacklist nf_conntrack_ftp
blacklist xt_state
blacklist iptable_nat
blacklist ipt_REDIRECT
blacklist nf_nat
blacklist nf_conntrack_ipv4
EOF
}

stop_service lbmanager
stop_service keepalived
stop_service ospfd
stop_service zebra

# disable conntrack
sed -i "s/--ip-masq//g" /usr/lib/systemd/system/flanneld.service
if [ -f /usr/bin/flanneld-start ];then
sed -i "s/--ip-masq//g" /usr/bin/flanneld-start
fi

disable_conntrack

# config syslog
touch /var/log/keepalived.log
touch /var/log/lbmanager.log
cat > /etc/rsyslog.d/lvs.conf <<EOF
\$ModLoad imudp
\$UDPServerRun 514
local2.*    /var/log/keepalived.log
local5.*    /var/log/lbmanager.log
EOF

sed -i "s:\*.info.*:\*.info;mail.none;authpriv.none;cron.none;local2.none;local5.none  /var/log/messages:" /etc/rsyslog.conf
sed -i "s/\*.emerg.*/\*.emerg;local2.none;local5.none                       \*/" /etc/rsyslog.conf

# modify journald
sed -i "/ForwardToSyslog/d" /etc/systemd/journald.conf
echo "ForwardToSyslog=yes" >> /etc/systemd/journald.conf
systemctl restart systemd-journald

systemctl restart rsyslog.service

bash $ROOT/conf_quagga.sh $UP_LINK $OSPF_AREA $OSPF_VIP $OSPF_ROUTE_FILTER
if [ $? -ne 0 ];then
    echo "install quagga fail"
    exit 1
fi

# 2. install kernel keepalived ipvsadm
yum clean all
yum install -y libnl python-IPy
rpm -ivh $RPM_ROOT/kernel-3.10.0-327.37.el7.lvs.x86_64.rpm --force
rpm -ivh $RPM_ROOT/keepalived-1.2.13-12.el7.lvs.x86_64.rpm --force
rpm -ivh $RPM_ROOT/ipvsadm-1.26-9.el7.lvs.x86_64.rpm --force
#yum install -y kernel keepalived ipvsadm --disablerepo=\* --enablerepo=xxxx

# 3. config boot grub
nohz=`grep nohz $GRUB_CONF`
if [ $? -ne 0 ];then
    sed -i 's/quiet/quiet nohz=off /' $GRUB_CONF
fi

grub2-mkconfig -o /boot/grub2/grub.cfg

# 4. install scripts
mkdir -p $LVS_ROOM
cp -uf $ROOT/conf_lvs.sh $LVS_ROOM
cp -uf $ROOT/udp_checker.py $LVS_ROOM
cp -uf $ROOT/bin/* $LVS_ROOM
cp -uf $ROOT/../system/nic.sh $LVS_ROOM
cp -uf $ROOT/lvs_discovery.py $LVS_ROOM
cp -uf $ROOT/lvs_stats_monitor.py $LVS_ROOM
chmod 755 $LVS_ROOM/*

# support mask beyond 24
local_ip_range=`python ${ROOT}/ip.py ${FLANNEL_SUBNET}`
if [ -z $local_ip_range ];then
    echo "get local ip range fail"
    exit 1
fi

second_ip_range=`python ${ROOT}/ip.py ${INCOMM_CIDR} 2>/dev/null`
if [ $? -ne 0 ];then
    second_ip_range=$UP_LINK_IP
fi

second_ip_mask=`echo ${INCOMM_CIDR} | awk -F"/" '{print $2}'`
if [ -z $second_ip_mask ];then
    second_ip_mask=24
fi

yum install -y bridge-utils
cat > /etc/sysconfig/lvs <<EOF
UP_LINK=${UP_LINK}
DOWN_LINK=${DOWN_LINK}
DOWN_LINK_IP=${FLANNEL_SUBNET}
DOWN_LINK_MTU=${FLANNEL_MTU}
LOCAL_IP_RANGE=${local_ip_range}
SECOND_IP_RANGE=${second_ip_range}
SECOND_IP_MASK=${second_ip_mask}
EOF

cat > /usr/lib/systemd/system/lvs.service <<EOF
[Unit]
Description=Config lvs after host reboot
After=flanneld.service
Wants=flanneld.service
Before=keepalived.service

[Service]
Type=oneshot
EnvironmentFile=/etc/sysconfig/lvs
ExecStart=${LVS_ROOM}/conf_lvs.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable lvs

bash $ROOT/conf_keepalived.sh $local_ip_range $second_ip_range
if [ -f /etc/sysconfig/keepalived ];then
    sed -i "s/KEEPALIVED_OPTIONS.*/KEEPALIVED_OPTIONS=\"-D -S 2\"/" /etc/sysconfig/keepalived
fi
cat > /usr/lib/systemd/system/keepalived.service <<EOF
[Unit]
Description=LVS and VRRP High Availability Monitor
After=syslog.target network.target lvs.service

[Service]
Type=forking
KillMode=process
EnvironmentFile=-/etc/sysconfig/keepalived
ExecStart=/usr/sbin/keepalived \$KEEPALIVED_OPTIONS
ExecReload=/bin/kill -HUP \$MAINPID
StandardOutput=syslog
SyslogFacility=local2

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable keepalived

# lvs stats
cron_file=/var/spool/cron/root
if [ ! -f "$cron_file" ]; then
    cat > $cron_file <<EOF
#SHELL=/bin/bash
#PATH=/sbin:/bin:/usr/sbin:/usr/bin

# For details see man 4 crontabs
#
# # Example of job definition:
# # .---------------- minute (0 - 59)
# # |  .------------- hour (0 - 23)
# # |  |  .---------- day of month (1 - 31)
# # |  |  |  .------- month (1 - 12) OR jan,feb,mar,apr ...
# # |  |  |  |  .---- day of week (0 - 6) (Sunday=0 or 7) OR sun,mon,tue,wed,thu,fri,sat
# # |  |  |  |  |
# # *  *  *  *  *  command to be executed
#
*/1 * * * * /opt/lvs/lvs_stats_monitor.py monitor
EOF
else
    if ! grep lvs_stats_monitor.py $cron_file > /dev/null 2>&1; then
        echo "*/1 * * * * /opt/lvs/lvs_stats_monitor.py monitor" >> $cron_file
    fi
fi
systemctl enable  crond.service
systemctl restart crond.service


bash $ROOT/install_lbmanager.sh ${ETCD_ENDPOINTS}

mkdir -p /etc/lb


#TODO add laddr group to be configed
cat > /etc/lb/lb.conf <<EOF
{
    "type": "l4",
    "store": "/xxx.com/",
    "cluster": "${CLUSTER_NAME}",
    "catalog": "dev",
    "confPath": "/etc/keepalived",
    "l4": {
        "subnet": "${SUBNAME}",
        "subnetCIDR": "${OSPF_VIP}",
        "keepalived": {
            "miscScript": "/opt/lvs/udp_checker.py"
        },
        "laddr": [
            {
                "name": "laddr_g1",
                "dest": "172.11.0.0/16"
            },
            {
                "name": "laddr_g2",
                "dest": "10.0.0.0/8"
            }
        ]
    }
}
EOF

sync
exit 0
