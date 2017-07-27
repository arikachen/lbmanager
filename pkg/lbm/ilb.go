package lbm

import "github.com/ArikaChen/lbmanager/pkg/leader"

type ILB interface {
	Parse(key string, data []byte) (interface{}, error)
	ConfigService(lb interface{}) bool
	DeleteService(name string) bool
	Reload()
	Recover() error
}

func New(t string) leader.IHandler {
	if t == "l4" {
		return NewLVSImpl()
	} else {
		return NewNginxImpl()
	}
}
