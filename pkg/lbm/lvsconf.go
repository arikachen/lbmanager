package lbm

import (
	"errors"

	"github.com/ArikaChen/lbmanager/pkg/utils"
)

const (
	LVSFeatureOptions = "lvs"

	maxTimeout = 10000
	maxBPS     = 10000
	maxCPS     = 20000
)

var (
	strategyOpts = []string{"rr", "wrr", "dh", "sh", "sed", "nq", "lc", "wlc", "lblc", "lblcr"}
	kindOpts     = []string{"FNAT"}
)

type LVSConf struct {
	Strategy           string `json:"algo,omitempty"`
	PersistenceTimeout uint   `json:"persistenceTimeout,omitempty"`
	SynProxy           bool   `json:"synProxy,omitempty"`
	Kind               string `json:"kind,omitempty"`
	BPSLimit           uint   `json:"bpsLimit,omitempty"`
	CPSLimit           uint   `json:"cpsLimit,omitempty"`
}

func (l *LVSConf) CheckValid() error {
	if l.Strategy != "" {
		if utils.IsElementExist(l.Strategy, strategyOpts) {
			return nil
		}
		return errors.New("algo is invalid")
	}
	if l.PersistenceTimeout > maxTimeout {
		return errors.New("persistence_timeout is invalid")
	}

	if l.Kind != "" {
		if utils.IsElementExist(l.Kind, kindOpts) {
			return nil
		}
		return errors.New("kind is invalid")
	}
	if l.BPSLimit > maxBPS {
		return errors.New("bps_limit is invalid")
	}
	if l.CPSLimit > maxCPS {
		return errors.New("cps_limit is invalid")
	}
	return nil
}
