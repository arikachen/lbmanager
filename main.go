package main

import (
	"flag"

	"github.com/ArikaChen/lbmanager/pkg/conf"
	"github.com/ArikaChen/lbmanager/pkg/kvstore"
	"github.com/ArikaChen/lbmanager/pkg/lbm"
	"github.com/ArikaChen/lbmanager/pkg/leader"

	"github.com/golang/glog"
)

var (
	endpoints = flag.String("etcd-endpoints", "", `etcd endponits, like: 127.0.0.1:2379,x.x.x.x:2379`)
)

func main() {
	flag.Parse()

	if *endpoints == "" {
		glog.Fatalf("Etcd endpoints is required")
	}

	conf.Init()

	kvstore.Init(*endpoints)
	kvstore.WaitForStart()

	lb := lbm.New(conf.Get().Type)

	leader.Run(lb)
}
