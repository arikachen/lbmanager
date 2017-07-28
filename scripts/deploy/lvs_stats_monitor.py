#!/usr/bin/python

import os
import sys
import commands

keepalive_paths = ["/etc/keepalived/conf.d/", "/etc/keepalived/common.conf.d/"]
suffix = ".conf"
svc_prefix = "virtual_server_group"
rs_prefix = "real_server"
stats_file = "/var/run/lvs_stats"
kinds = ['conns', 'inpkts', 'outpkts', 'inbytes', 'outbytes', 'cps', 'inpps', 'outpps', 'inbps', 'outbps']

def get_svc(conf_path):
    svc = {}
    for _, _, f in os.walk(conf_path):
        for it in f:
            if it.endswith(suffix):
                s = it.split('.')[0]
                svc_url = ""
                rsl = []
                with open(conf_path + it) as fh:
                    for l in fh:
                        if l.startswith(svc_prefix):
                            svc_url = l.split(' ')[1]
                        if rs_prefix in l:
                            rs = l.split(' ')
                            rs_url = rs[1] + ':' + rs[2]
                            if (len(rs_url.split(':')) is 2) and (len(svc_url) != 0):
                                rsl.append(rs_url)
                if len(rsl) != 0:
                    svc[svc_url] = rsl
    return svc


def exec_ipvsadm(svc, type):
    cmd_fmt = "/usr/sbin/ipvsadm -ln --%s -t %s"

    cmd = cmd_fmt % (type, svc)
    return commands.getstatusoutput(cmd)

def parse_data(svc, output):
    output = output.split('\n')
    if len(output) < 3:
        return None
    sl = []
    for i in range(2, len(output)):
       data =  output[i].split()
       data[0] = svc
       sl.append(data)
    return sl

def get_stats(svc):
    _, output = exec_ipvsadm(svc, 'stats')
    return parse_data(svc, output)

def get_rate(svc):
    _, output = exec_ipvsadm(svc, 'rate')
    return parse_data(svc, output)

def parse_stats(it):
    if it.endswith('K'):
        return it.replace('K', '000')
    elif it.endswith('M'):
        return it.replace('M', '000000')
    elif it.endswith('G'):
        return it.replace('G', '000000000')
    elif it.endswith('T'):
        return it.replace('T', '000000000000')
    return it

def monitor_stats():
    svc = {}
    for _, v in enumerate(keepalive_paths):
        svc = dict(svc, **get_svc(v))
    ss = []
    for k, v in svc.items():
        s = get_stats(k)
        r = get_rate(k)
        if ((s and r) is not None) and (len(s) is len(r)):
           for i in range(len(s)):
               if s[i][0:2] == r[i][0:2]:
                  t = s[i]
                  t.extend(r[i][2:])
                  ss.append(' '.join(t) + '\n')
    with open(stats_file, 'w') as f:
       f.writelines(ss)

def trigger_stats(svc, filter, kind):
    with open(stats_file) as f:
        for l in f:
            data = l.split()
            if len(data) != len(kinds) + 2:
                continue
            if data[0:2] == [svc, filter]:
                return data[kinds.index(kind) + 2]
    return None


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print "wrong argv num"
        exit(1)
    op = sys.argv[1]
    opl = ['monitor', 'get']
    if op not in opl:
        print "invalid operation"
        exit(1)

    if op == opl[0]:
        monitor_stats()
    else:
        if len(sys.argv) < 5:
            print "wrong argv num"
            exit(1)
        svc = sys.argv[2]
        real = sys.argv[3]
        kind = sys.argv[4]
        try:
            stats = trigger_stats(svc, real, kind)
            if stats is None:
                exit(1)
            else:
                print parse_stats(stats)
        except:
            print 0
            exit(1)
