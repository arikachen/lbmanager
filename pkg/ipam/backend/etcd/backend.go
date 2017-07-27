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

package etcd

import (
	"net"

	"github.com/ArikaChen/lbmanager/pkg/ipam/backend"
	"github.com/ArikaChen/lbmanager/pkg/kvstore"
)

const (
	lastIPFile = "last_reserved_ip"
)

type Store struct {
	item string
}

// Store implements the Store interface
var _ backend.Store = &Store{}

func New(network, dataDir string) (*Store, error) {
	found, err := kvstore.IsExist(dataDir, network)
	if err != nil {
		return nil, err
	}
	if !found {
		err := kvstore.WriteDir(dataDir, network)
		if err != nil {
			return nil, err
		}
	}
	return &Store{item: dataDir + network + "/"}, nil
}

func (s *Store) Reserve(id string, ip net.IP) (bool, error) {
	found, err := kvstore.IsExist(s.item, ip.String())
	if err != nil {
		return false, err
	}
	if found {
		return false, nil
	}

	err = kvstore.Write(s.item, ip.String(), id)
	if err != nil {
		return false, err
	}
	err = kvstore.Write(s.item, lastIPFile, ip.String())
	if err != nil {
		return false, err
	}
	return true, nil
}

// LastReservedIP returns the last reserved IP if exists
func (s *Store) LastReservedIP() (net.IP, error) {
	ip, err := kvstore.Read(s.item, lastIPFile)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(ip), nil
}

func (s *Store) Release(ip net.IP) error {
	return kvstore.Delete(s.item, ip.String())
}

// N.B. This function eats errors to be tolerant and
// release as much as possible
func (s *Store) ReleaseByID(id string) error {
	ip, err := s.Exist(id)
	if err != nil {
		return err
	}
	if ip != nil {
		return s.Release(ip)
	}
	return nil
}

func (s *Store) Exist(id string) (net.IP, error) {
	key, err := kvstore.Exist(s.item, id)
	if err != nil {
		return nil, err
	}
	if key != "" {
		return net.ParseIP(key), nil
	}
	return nil, nil
}

func (s *Store) Recover() (map[string]string, error) {
	l, err := kvstore.List(s.item)
	m := make(map[string]string)
	if err != nil {
		return m, err
	}
	for _, it := range l {
		if it.Key == lastIPFile {
			continue
		}
		m[string(it.Value)] = it.Key
	}
	return m, nil
}

func (s *Store) Lock() error {
	return nil
}

func (s *Store) Unlock() error {
	return nil
}

func (s *Store) Close() error {
	return nil
}
