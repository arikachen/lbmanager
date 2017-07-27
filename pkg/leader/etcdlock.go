package leader

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ArikaChen/lbmanager/pkg/conf"
	"github.com/ArikaChen/lbmanager/pkg/kvstore"
	"github.com/docker/libkv/store"

	kerros "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

const Item = "election"

type ETCDLock struct {
	Item       string
	Key        string
	LockConfig rl.ResourceLockConfig
	pair       *store.KVPair
}

// Get returns the cmlection record from a ConfigMap Annotation
func (l *ETCDLock) Get() (*rl.LeaderElectionRecord, error) {
	var record rl.LeaderElectionRecord
	var err error
	l.pair, err = kvstore.ReadObj(l.Item, l.Key)
	if err != nil {
		if err == store.ErrKeyNotFound {
			return nil, kerros.NewNotFound(schema.GroupResource{
				Group:    l.Item,
				Resource: l.Key,
			}, l.Key)
		}
		return nil, err
	}
	if err := json.Unmarshal(l.pair.Value, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// Create attempts to create a LeadercmlectionRecord annotation
func (l *ETCDLock) Create(ler rl.LeaderElectionRecord) error {
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return err
	}
	_, l.pair, err = kvstore.WriteAtomic(l.Item, l.Key, recordBytes, nil, nil)
	return err
}

// Update will update and existing annotation on a given resource.
func (l *ETCDLock) Update(ler rl.LeaderElectionRecord) error {
	if l.pair == nil {
		return errors.New("hb not initialized, call get or create first")
	}
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return err
	}
	_, l.pair, err = kvstore.WriteAtomic(l.Item, l.Key, recordBytes, l.pair, nil)
	return err
}

// RecordEvent in leader cmlection while adding meta-data
func (l *ETCDLock) RecordEvent(s string) {
	//events := fmt.Sprintf("%v %v", l.LockConfig.Identity, s)
}

// Describe is used to convert details on current resource lock
// into a string
func (l *ETCDLock) Describe() string {
	return fmt.Sprintf("%v%v", l.Item, l.Key)
}

// returns the Identity of the lock
func (l *ETCDLock) Identity() string {
	return l.LockConfig.Identity
}

func New(name string, rlc rl.ResourceLockConfig) (rl.Interface, error) {
	lm := &ETCDLock{
		Item:       fmt.Sprintf("%s/%s/", Item, conf.Get().Catalog),
		Key:        name,
		LockConfig: rlc,
	}
	return lm, nil
}
