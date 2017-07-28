#!/usr/bin/python

import sys
from IPy import IP

def get_ip_range(subnet):
    ips = subnet.split('/')
    if len(ips) != 2 or int(ips[1]) < 24:
        return ""
    ip = IP(ips[0]).make_net(ips[1])
    l = str(ip).split('/')[0].split('.')
    s = int(l[3]) + 2
    e = int(l[3]) + len(ip) - 1 - 1
    l[3] = str(s)
    if e < s:
        return ""
    elif e == s:
        return '.'.join(l)
    else:
        return '.'.join(l) + "-" + str(e)

sub = sys.argv[1]
print get_ip_range(sub)
