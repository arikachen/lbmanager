#!/usr/bin/python

import os
import json

keepalive_path = "/etc/keepalived/conf.d/"
suffix = ".conf"
svc_prefix = "virtual_server_group"
rs_prefix = "real_server"

def get_svc():
    svc = []
    for _, _, f in os.walk(keepalive_path):
        for it in f:
            if it.endswith(suffix):
                s = it.split('.')[0]
                svc_url = ""
		with open(keepalive_path + it) as fh:
                    for l in fh:
                        if l.startswith(svc_prefix):
                            svc_url = l.split(' ')[1]
                            if len(svc_url.split(':')) is 2:
                                svc += [{'{#SVCNAME}' : s, '{#SVCURL}' : svc_url}]
                        if rs_prefix in l:
                            rs = l.split(' ')
                            rs_url = rs[1] + ':' + rs[2]
                            if rs[1] == '127.0.0.1':
                                continue
                            if (len(rs_url.split(':')) is 2) and (len(svc_url) != 0):
                                svc += [{'{#SVCPARENT}' : svc_url, '{#RSURL}' : rs_url}]
    return svc


a = get_svc()
print json.dumps({'data':a},sort_keys=True,indent=7,separators=(',',':'))
