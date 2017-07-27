package lbm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/ArikaChen/lbmanager/pkg/conf"
	"github.com/ArikaChen/lbmanager/pkg/kvstore"

	"github.com/docker/libkv/store"
	"github.com/golang/glog"
	"github.com/pborman/uuid"
)

const (
	item = "lb/"
)

type LBBase struct {
	Conf       conf.LBConf
	ConfigPath string
	tmplBuf    *bytes.Buffer

	Item string
	Key  string

	IsLeader bool
	Stop     chan struct{}

	Elem map[string]interface{}
	Impl ILB
}

func NewLB() LBBase {
	c := conf.Get()
	lb := LBBase{
		Conf:    c,
		Item:    fmt.Sprintf("%s/%s/", c.Catalog, c.Type),
		tmplBuf: bytes.NewBuffer(make([]byte, 0, defBufferSize)),
	}
	lb.Key = item + lb.Item
	return lb
}

func (b *LBBase) Render(svcPath, tempDef string, obj interface{}) (bool, error) {
	defer b.tmplBuf.Reset()

	tmpName := obj.(ILBCommon).GetName()
	tmpl, err := template.New(tmpName).Parse(tempDef)
	if err != nil {
		glog.Errorf("Failed to parse template file, service: %s, error: %v", tmpName, err)
		return false, err
	}

	if err = tmpl.Execute(b.tmplBuf, obj); err != nil {
		glog.Errorf("Failed to fill template, error: %v", err)
		return false, err
	}
	content := b.tmplBuf.Bytes()

	src, err := ioutil.ReadFile(svcPath)
	if err != nil && !strings.Contains(err.Error(), errFileNotExist) {
		glog.Errorf("Failed to read service config file, service: %s, error: %v", tmpName, err)
		return false, err
	}

	if err == nil && bytes.Equal(src, content) {
		glog.V(3).Infof("Config is not modified, service: %s", tmpName)
		return false, nil
	}

	glog.V(3).Infof("Writing conf to %s", svcPath)
	err = ioutil.WriteFile(svcPath, content, 0644)
	if err != nil {
		glog.Errorf("Write file %s failed, service: %s, error: %v", svcPath, tmpName, err)
		return false, err
	}

	glog.Infof("%s", string(content))
	glog.Infof("LB service %s had been updated", tmpName)
	return true, nil
}

func (b *LBBase) IsSameCluster(lb interface{}) bool {
	return lb.(ILBCommon).GetClusterName() == b.Conf.ClusterName
}

func (b *LBBase) GetConfigFilePath(name string) string {
	return path.Join(b.ConfigPath, name+".conf")
}

func (b *LBBase) handlerAddOrUpdate(li []*store.KVPair) bool {
	var lbs []interface{}
	for _, it := range li {
		lb, err := b.Impl.Parse(it.Key, it.Value)
		if err != nil {
			continue
		}
		if !b.IsSameCluster(lb) {
			continue
		}
		kArr := strings.Split(it.Key, "/")
		if kArr[len(kArr)-1] != lb.(ILBCommon).GetName() {
			glog.Warningf("Key %s is mismatch with lb name %s", kArr[len(kArr)-1], lb.(ILBCommon).GetName())
			continue
		}

		obj, ok := b.Elem[it.Key]
		if ok && reflect.DeepEqual(lb, obj) {
			continue
		}

		lbs = append(lbs, lb)
		b.Elem[it.Key] = lb
		glog.V(4).Infof("Add or update key %s, val %v", it.Key, lb)
	}

	needReload := false
	for _, lb := range lbs {
		if b.Impl.ConfigService(lb) {
			needReload = true
		}
	}
	return needReload
}

func (b *LBBase) handlerDelete(li []*store.KVPair) bool {
	var lbs []string
	for k, v := range b.Elem {
		found := false
		for _, it := range li {
			if k == it.Key {
				found = true
				break
			}
		}
		if !found {
			lbs = append(lbs, v.(ILBCommon).GetName())
			delete(b.Elem, k)
			glog.V(4).Infof("Delete key %s", k)
		}
	}
	needReload := false
	for _, lb := range lbs {
		if b.Impl.DeleteService(lb) {
			needReload = true
		}
	}
	return needReload
}

func (b *LBBase) handlerEvent(li []*store.KVPair) {
	needReload := b.handlerAddOrUpdate(li)
	if b.handlerDelete(li) {
		needReload = true
	}
	if needReload {
		b.Impl.Reload()
	}
}

func (b *LBBase) watch(id string, stopCh <-chan struct{}) {
	for {
		glog.Infof("Start watching %s lb, key: %s, id: %s", b.Conf.Type, b.Key, id)
		found, err := kvstore.IsExist(item, b.Item)
		if err != nil {
			glog.Warningf("Failed to get lb dir, error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		} else if !found {
			err = kvstore.WriteDir(item, b.Item)
			if err != nil {
				glog.Warningf("Failed to write %s dir, error: %v", b.Item, err)
				time.Sleep(1 * time.Second)
				continue
			}
		}
		events, err := kvstore.WatchTree(item, b.Item, stopCh)
		if err != nil || events == nil {
			glog.Warningf("Watch %s lb fail, retry", b.Item)
			time.Sleep(2 * time.Second)
			continue
		}

		handler := func() {
			for {
				select {
				case event := <-events:
					if event == nil {
						glog.Warningf("Watch %s event is nil, id: %s", b.Item, id)
						return
					}
					b.handlerEvent(event)
				case <-time.After(4 * time.Second):
					continue
				}
			}
		}
		handler()

		_, ok := <-stopCh
		if !ok {
			glog.Infof("Stop watching %s lb, key: %s, id: %s", b.Conf.Type, b.Key, id)
			return
		}
	}
}

func (b *LBBase) Run(isLeader bool, stopCh <-chan struct{}) {
	id := uuid.New()
	glog.Infof("Start %s lb, id: %s", b.Conf.Type, id)
	//role changed
	if isLeader && b.Stop != nil {
		close(b.Stop)
		time.Sleep(1 * time.Second)
	}

	b.Impl.Recover()
	b.IsLeader = isLeader

	stop := make(chan struct{})

	go b.watch(id, stop)

	func() {
		for {
			select {
			case <-stopCh:
				glog.Infof("Stop %s lb, id: %s", b.Conf.Type, id)
				return
			default:
			}
		}
	}()

	close(stop)
	time.Sleep(1 * time.Second)
}

func (b *LBBase) Change() {
	b.Stop = make(chan struct{})
	b.Run(false, b.Stop)
}

func (b *LBBase) Init() {
}
