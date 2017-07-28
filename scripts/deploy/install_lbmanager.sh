#!bin/bash

ETCD_ENDPOINTS=$1
#install lbmanager
cat > /etc/sysconfig/lbmanager <<EOF
ETCD_ENDPOINTS=${ETCD_ENDPOINTS}
OPTIONS="--v=2"
EOF

cat > /usr/lib/systemd/system/lbmanager.service <<EOF
[Unit]
Description=lb manager
After=lvs.service keepalived.service

[Service]
Type=simple
EnvironmentFile=/etc/sysconfig/lbmanager
StandardOutput=syslog
SyslogFacility=local5
ExecStart=/opt/lvs/lbmanager -etcd-endpoints \${ETCD_ENDPOINTS}  \\
        \${OPTIONS}

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable lbmanager

