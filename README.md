# lbmanager

Simple loadbalance manager. It is support L4 and L7.
* **L4**: ali [LVS][LVS] with FULLNAT, 
* **L7**: [nginx][nginx]


[LVS]: https://github.com/alibaba/LVS
[nginx]: https://github.com/nginx/nginx

## Getting Started with L4

### Building LVS
It it base on centos 7.2, you should config yum repo first.
```sh
git clone https://github.com/ArikaChen/lbmanager.git
cd build
sh build.sh
```

### Building lbmanager
```sh
go build
```

### Deploy
using ansible playbook to deploy lbmanager.
* install [etcd][etcd]
* install [flannel][flannel] first, using network 172.11.0.0/16, the case is using 
  flannel to connect with backend
* copy rpms to dest host, if you upload the rpm to a repo, ignore this step
* change the scripts/deploy/deploy.sh#RPM_ROOT to rpm dir
* copy lbmanager binary to scripts/deploy/bin
* modify scripts/ansible/clusters/test-l4-cluster with your env

[etcd]: https://github.com/coreos/etcd/blob/master/Documentation/op-guide/clustering.md
[flannel]: https://github.com/coreos/flannel/blob/master/Documentation/running.md

```sh
ansible-playbook -i clusters/test-l4-cluster lb.yaml --tags l4 --tags deploy
```

**Notice** Operation will reboot the host

### Create
```sh
etcdctl --endpoints http://127.0.0.1:2379 mk /xxx.com/lb/dev/l4/test \
'{"name":"test","cluster":"lvs-cluster","servers":[{"ip":"172.11.20.20","port":50057}]}'
```

### Check
```sh
ipvsadm -ln
```


## TODO

### LVS
* use Linux HTB to enhance SLA
* use ipset to do whitelist or blacklist
* LVS DPDK?

### lbmanager
* add update status support
* enhance nginx config and deploy
* add l7 policy support, struct is similar with openstack [octavia][octavia]
* work with k8s [ingress controller][ingress]
* ...

[octavia]: https://github.com/openstack/octavia
[ingress]: https://github.com/kubernetes/ingress
