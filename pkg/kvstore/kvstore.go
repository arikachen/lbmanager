package kvstore

import (
	"strings"
	"time"

	"github.com/ArikaChen/lbmanager/pkg/conf"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/etcd"
	"github.com/golang/glog"
)

var (
	kv kvStore
)

type kvStore struct {
	store  store.Store
	prefix string
}

func Init(clients string) {
	etcd.Register()
	c := strings.Split(clients, ",")
	s, err := libkv.NewStore(store.ETCD, c, nil)
	if err != nil {
		glog.Fatal("Init kvstore failed, error: ", err)
	}

	kv = kvStore{store: s, prefix: conf.GetStorePrefix()}
	glog.Infof("Init kvstore, kvstore is %s", conf.GetStorePrefix())
}

func WaitForStart() {
	for {
		_, err := kv.store.List(kv.prefix)
		if err == nil || err == store.ErrKeyNotFound {
			return
		}

		glog.Warningf("KVStore is not started, error: %v", err)
		time.Sleep(2 * time.Second)
	}
}

func Write(item, key, val string) error {
	return kv.store.Put(GetStoreKey(item, key), []byte(val), nil)
}

func WriteDir(item, key string) error {
	return kv.store.Put(GetStoreKey(item, key), []byte{}, &store.WriteOptions{IsDir: true})
}

// now only support read a string
func Read(item, key string) (string, error) {
	pair, err := kv.store.Get(GetStoreKey(item, key))
	if err != nil {
		return "", err
	}
	return string(pair.Value), nil
}

func ReadObj(item, key string) (*store.KVPair, error) {
	return kv.store.Get(GetStoreKey(item, key))
}

func Delete(item, key string) error {
	return kv.store.Delete(GetStoreKey(item, key))
}

func Exist(item, val string) (string, error) {
	pairs, err := kv.store.List(GetStoreKey(item, ""))
	if err != nil {
		return "", err
	}
	for _, it := range pairs {
		if val == string(it.Value) {
			lk := strings.Split(it.Key, "/")
			return lk[len(lk)-1], nil
		}
	}
	return "", nil
}

func IsExist(item, key string) (bool, error) {
	return kv.store.Exists(GetStoreKey(item, key))
}

func WatchTree(item, key string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	return kv.store.WatchTree(GetStoreKey(item, key), stopCh)
}

func WriteBytes(item, key string, val []byte) error {
	return kv.store.Put(GetStoreKey(item, key), val, nil)
}

func List(key string) ([]*store.KVPair, error) {
	return kv.store.List(GetStoreKey("", key))
}

func GetStoreKey(item, key string) string {
	return kv.prefix + item + key
}

func WriteAtomic(item, key string, val []byte, previous *store.KVPair, opts *store.WriteOptions) (bool, *store.KVPair, error) {
	return kv.store.AtomicPut(GetStoreKey(item, key), val, previous, opts)
}
