package lbm

import (
	"encoding/json"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ArikaChen/lbmanager/pkg/conf"
	"github.com/ArikaChen/lbmanager/pkg/ipam"
	"github.com/ArikaChen/lbmanager/pkg/keepalived"
	"github.com/ArikaChen/lbmanager/pkg/kvstore"
	"github.com/ArikaChen/lbmanager/pkg/stats"
	"github.com/ArikaChen/lbmanager/pkg/utils"

	"github.com/golang/glog"
)

const (
	stick = "l4-"
)

type LVSImpl struct {
	LBBase

	Stick string
}

type LVSInner struct {
	LVS

	LAddrGroup string
	KConf      conf.Keepalived
}

func NewLVSImpl() *LVSImpl {
	lm := &LVSImpl{
		LBBase: NewLB(),
		Stick:  stick,
	}
	lm.ConfigPath = path.Join(lm.Conf.ConfDir, "conf.d")
	lm.Elem = make(map[string]interface{})
	lm.Impl = lm
	return lm
}

func (l *LVSImpl) Parse(key string, data []byte) (interface{}, error) {
	lb := NewLVS()
	err := json.Unmarshal(data, lb)
	if err != nil {
		glog.Warningf("Faild to unmarshal lb %s, error: %v", key, err)
		return nil, err
	}

	err = lb.Validate()
	if err != nil {
		glog.Warningf("LB %s request is invalid, error: %v", key, err)
		return nil, err
	}

	return lb, nil
}

func (l *LVSImpl) Reload() {
	keepalived.Reload()
}

func (l *LVSImpl) allocVIP(lvs *LVS) error {
	update := false
	name := lvs.Name
	vip := lvs.VIP
	ip, err := ipam.RequireIP(l.Stick+name, vip)
	if err != nil {
		glog.Errorf("Failed to alloc vip, name is %s", name)
		return err
	}

	defer func() {
		if err != nil && vip != ip {
			ipam.ReleaseIP(l.Stick + name)
		}
	}()

	update = (vip != ip)
	lvs.VIP = ip
	if lvs.Port == 0 {
		s := strings.Split(ip, ".")
		if len(s) == 4 {
			p, _ := strconv.Atoi(s[3])
			lvs.Port = l.Conf.L4Conf.PortBase + p
			update = true
		}
	}
	if update {
		glog.Infof("Update lb, name is %s, vip is %s, port is %d", name, lvs.VIP, lvs.Port)
		err := lvs.Update(l.Key)
		if err != nil {
			glog.Errorf("Failed to update lvs, name is %s", name)
			return err
		}
	}
	return nil
}

func (l *LVSImpl) checkValid(lvs *LVS) bool {
	if lvs.VIP == "" {
		glog.Warningf("Virtual IP is required, lb: %s", lvs.Name)
		return false
	}
	if lvs.Port == 0 {
		glog.Errorf("Virtual Port %d is invalid, lb: %s", lvs.Port, lvs.Name)
		return false
	}
	return true
}

func (l *LVSImpl) ConfigService(lb interface{}) bool {
	lvs := lb.(*LVS)
	glog.Infof("Updating lvs configuration, %s", lvs.Name)
	if l.IsLeader {
		err := l.allocVIP(lvs)
		if err != nil {
			return false
		}
	} else {
		if !l.checkValid(lvs) {
			return false
		}
	}
	return l.renderTemplate(lvs)
}

func (l *LVSImpl) DeleteService(name string) bool {
	if l.IsLeader {
		err := ipam.ReleaseIP(l.Stick + name)
		if err != nil {
			glog.Warningf("Failed to release %s vip, error: %v", name, err)
		}
	}

	filename := l.GetConfigFilePath(name)
	if isExist, _ := utils.IsFileExist(filename); isExist {
		glog.Infof("Delete lb file %v", filename)
		if err := os.Remove(filename); err != nil {
			glog.Warningf("Failed to delete %v: %v", filename, err)
			return false
		}
		return true
	}
	return false
}

func (l *LVSImpl) renderTemplate(lvs *LVS) bool {
	defer l.tmplBuf.Reset()
	inner := &LVSInner{
		LVS:   *lvs,
		KConf: l.Conf.L4Conf.KConf,
	}
	if len(lvs.Servers) > 0 {
		inner.LAddrGroup, _ = conf.GetLAddrGroupName(lvs.Servers[0].IP)
	}

	fileName := l.GetConfigFilePath(lvs.Name)
	needReload, _ := l.Render(fileName, lvsTmpl, inner)
	return needReload
}

func (l *LVSImpl) Recover() error {
	keys := ipam.GetKeys()
	for _, v := range keys {
		if strings.HasPrefix(v, l.Stick) {
			lb := NewLVS()
			lb.Name = strings.Replace(v, l.Stick, "", -1)
			lb.ClusterName = l.Conf.ClusterName
			l.Elem[kvstore.GetStoreKey(l.Key, lb.Name)] = lb
		}
	}
	glog.V(4).Infof("LVS recover, %v", l.Elem)
	return nil
}

func (l *LVSImpl) Init() {
	ipam.Init(l.Conf.L4Conf.SubnetName, l.Conf.L4Conf.SubnetCIDR)

	keepalived.Start()
	stats.Start()
}
