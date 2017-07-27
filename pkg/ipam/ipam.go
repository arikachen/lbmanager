// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipam

import (
	"net"
	"sync"

	"github.com/ArikaChen/lbmanager/pkg/ipam/backend/allocator"
	"github.com/ArikaChen/lbmanager/pkg/ipam/backend/etcd"
	"github.com/ArikaChen/lbmanager/pkg/ipam/types"

	"github.com/golang/glog"
)

var (
	dataDir = "ipam/"
	alc     *allocator.IPAllocator
	lock    sync.Mutex

	ipamCache = make(map[string]string)
)

func Init(name, cidr string) {
	n, err := types.ParseCIDR(cidr)
	if err != nil {
		glog.Fatal("Failed to init ipam, error: ", err)
	}
	ipamConf := allocator.NewIPAMConf(name, n)

	store, err := etcd.New(ipamConf.Name, dataDir)
	if err != nil {
		glog.Fatal("Failed to init ipam, error: ", err)
	}

	alc, err = allocator.NewIPAllocator(ipamConf, store)
	if err != nil {
		store.Close()
		glog.Fatal("Failed to init ipam, error: ", err)
	}
	ipamCache, err = alc.Recover()
	if err != nil {
		store.Close()
		glog.Fatal("Failed to recover ipam, error: ", err)
	}
}

func RequireIP(name string, reqIP string) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	glog.V(3).Infof("Require ip, name is %s, reqip is %s", name, reqIP)
	oldIP, err := alc.GetExist(name)
	if err != nil {
		glog.Errorf("Failed to require ip, error: %v", err)
		return "", err
	}

	if oldIP != nil && oldIP.String() != "" {
		if reqIP != "" && reqIP != oldIP.String() {
			alc.Release(name)
		} else {
			return oldIP.String(), nil
		}
	}
	ip := net.ParseIP(reqIP)
	ipConf, _, err := alc.Get(name, ip, true)
	if err != nil {
		glog.Errorf("Failed to require ip, error: %v", err)
		return "", err
	}
	glog.Infof("Require ip success, name is %s, ip is %s", name, ipConf.Address.IP.String())
	ipamCache[name] = ipConf.Address.IP.String()
	return ipConf.Address.IP.String(), nil
}

func ReleaseIP(name string) error {
	lock.Lock()
	defer lock.Unlock()
	glog.V(3).Infof("Release ip, name is %s", name)
	err := alc.Release(name)
	if err != nil {
		glog.Errorf("Failed to release ip, name: %s, error: %s", name, err)
		return err

	}
	delete(ipamCache, name)
	return nil
}

func GetKeys() []string {
	lock.Lock()
	defer lock.Unlock()
	var keys []string
	for k := range ipamCache {
		keys = append(keys, k)
	}
	return keys
}
