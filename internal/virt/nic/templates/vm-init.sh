#!/bin/bash

# initialize NIC
# usage:
# vm-init.sh ip1 gw1 ip2 gw2
ifs=$(ip l | grep state | awk -F ': ' '{ if($2 != "lo" ) {print $2} }')

for ifname in $ifs
do
    ip_addr=$1
    gw_addr=$2

    network="/etc/systemd/network/10-$ifname.network"
    cat << EOF > $network
[Match]
Name=$ifname

[Network]
Gateway=$gw_addr

[Address]
Address=$ip_addr
EOF
    shift 2
done

systemctl restart systemd-networkd

# prepare dns if neccessary
dnsOutput=$(dig +short baidu.com)
if [ -z "$dnsOutput" ]
then
    echo "Setting DNS..."
    mkdir /etc/systemd/resolved.conf.d/
    cat << EOF > /etc/systemd/resolved.conf.d/dns_servers.conf
[Resolve]
DNS=8.8.8.8 1.1.1.1
EOF

    systemctl restart systemd-resolved
fi