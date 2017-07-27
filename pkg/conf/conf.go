package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/ArikaChen/lbmanager/pkg/utils"

	"github.com/golang/glog"
)

const (
	confPath     = "/etc/lb/lb.conf"
	lvsStatsPort = 33611
	lvsStatsPath = "/var/run/lvs_stats"
)

var (
	conf   LBConf
	lbType = []string{"l4", "l7"}
	laddr  = make(map[string]*net.IPNet)
)

type Keepalived struct {
	ConfDir    string `json:"confPath,omitempty"`
	ReloadTime int    `json:"reloadPeriod,omitempty"`
	MiscScript string `json:"miscScript,omitempty"`
}

func (k *Keepalived) CheckValid() error {
	if k.ReloadTime <= 0 {
		return fmt.Errorf("reloadPeriod %d is invalid", k.ReloadTime)
	}

	if _, err := utils.IsFileExist(k.MiscScript); err != nil {
		return fmt.Errorf("keepalived script path %s is invalid", k.MiscScript)
	}

	return nil
}

type LVSStats struct {
	ResultPath string `json:"path,omitempty"`
	Port       int    `json:"listen,omitempty"`
}

func (s *LVSStats) CheckValid() error {
	if !strings.HasPrefix(s.ResultPath, "/") {
		return fmt.Errorf("stats path %s is invalid", s.ResultPath)
	}

	if s.Port > 65535 || s.Port <= 0 {
		return fmt.Errorf("stats port %d is invalid", s.Port)
	}
	return nil
}

type LocalAddrConf struct {
	Name     string `json:"name,omitempty"`
	DestCIDR string `json:"dest,omitempty"`
}

func (l *LocalAddrConf) CheckValid() error {
	if l.Name == "" {
		return fmt.Errorf("lb addr name is required")
	}
	_, cidr, err := net.ParseCIDR(l.DestCIDR)
	if err != nil {
		return fmt.Errorf("lb addr dest cidr %s is invalid", l.DestCIDR)
	}
	laddr[l.Name] = cidr
	return nil
}

type L4Conf struct {
	SubnetName string          `json:"subnet,omitempty"`
	SubnetCIDR string          `json:"subnetCIDR,omitempty"`
	PortBase   int             `json:"portBase,omitempty"`
	KConf      Keepalived      `json:"keepalived,omitempty"`
	Stats      LVSStats        `json:"stats,omitempty"`
	LAddr      []LocalAddrConf `json:"laddr,omitempty"`
}

func (l *L4Conf) CheckValid() error {
	var err error
	if l.SubnetName == "" {
		return errors.New("lb subnet name is empty")
	}
	_, _, err = net.ParseCIDR(l.SubnetCIDR)
	if err != nil {
		return fmt.Errorf("lb subnet cidr %s is invalid", l.SubnetCIDR)
	}

	if l.PortBase > 65535 || l.PortBase < 5000 {
		return fmt.Errorf("lb port base %d is invalid", l.PortBase)
	}
	err = l.KConf.CheckValid()
	if err != nil {
		return err
	}

	err = l.Stats.CheckValid()
	if err != nil {
		return err
	}
	if len(l.LAddr) == 0 {
		return fmt.Errorf("lb laddr is required")
	}
	for _, v := range l.LAddr {
		err = v.CheckValid()
		if err != nil {
			return err
		}
	}
	return nil
}

type LBConf struct {
	Type        string `json:"type,omitempty"`
	StorePrefix string `json:"store,omitempty"`
	ClusterName string `json:"cluster,omitempty"`
	Catalog     string `json:"catalog,omitempty"`
	ConfDir     string `json:"confPath,omitempty"`
	L4Conf      L4Conf `json:"l4,omitempty"`
}

func (c *LBConf) CheckValid() error {
	if !utils.IsElementExist(c.Type, lbType) {
		return fmt.Errorf("lb type %s is invalid", c.Type)
	}
	if !strings.HasPrefix(c.StorePrefix, "/") {
		return fmt.Errorf("lb store %s is invalid", c.StorePrefix)
	}

	if c.ClusterName == "" {
		return errors.New("cluster name is empty")
	}

	if _, err := utils.IsFileExist(c.ConfDir); err != nil {
		return fmt.Errorf("lb conf path %s is invalid", c.ConfDir)
	}

	if c.Type == "l4" {
		return c.L4Conf.CheckValid()
	}
	return nil
}

func Init() {
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		glog.Fatal("Failed to read lb conf file, error: ", err)
	}
	lb := LBConf{
		L4Conf: L4Conf{
			PortBase: 10000,
			KConf: Keepalived{
				ReloadTime: 5,
			},
			Stats: LVSStats{
				ResultPath: lvsStatsPath,
				Port:       lvsStatsPort,
			},
		},
	}
	err = json.Unmarshal(data, &lb)
	if err != nil {
		glog.Fatal("Failed to unmarshal lb conf file, error: ", err)
	}
	err = lb.CheckValid()
	if err != nil {
		glog.Fatal("LB config file is invalid, error: ", err)
	}
	conf = lb
}

func Get() LBConf {
	return conf
}

func GetStorePrefix() string {
	return conf.StorePrefix
}

func GetLAddrGroupName(ip string) (string, error) {
	back := net.ParseIP(ip)
	for k, v := range laddr {
		if v.Contains(back) {
			return k, nil
		}
	}
	return "", fmt.Errorf("can not find matched local addr group, ip: %s", ip)
}
