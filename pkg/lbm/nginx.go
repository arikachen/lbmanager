package lbm

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/ArikaChen/lbmanager/pkg/utils"

	"github.com/golang/glog"
)

const (
	cmd           = "/usr/sbin/nginx"
	checkPattern  = "%s -t"
	reloadPattern = "%s -s reload"
	pidFile       = "/run/nginx.pid"
)

type NginxImpl struct {
	LBBase

	streamConfPath string
}

func NewNginxImpl() *NginxImpl {
	lm := &NginxImpl{
		LBBase: NewLB(),
	}
	lm.ConfigPath = path.Join(lm.Conf.ConfDir, "conf.d")
	lm.streamConfPath = path.Join(lm.Conf.ConfDir, "stream.conf.d")
	lm.Elem = make(map[string]interface{})
	lm.Impl = lm
	return lm
}

func (n *NginxImpl) Parse(key string, data []byte) (interface{}, error) {
	lb := NewNginx()
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

func (n *NginxImpl) GetConfigFilePath(tp, name string) string {
	filePath := n.ConfigPath
	if tp != "HTTP" {
		filePath = n.streamConfPath
	}
	return path.Join(filePath, name+".conf")
}

func (n *NginxImpl) ConfigService(lb interface{}) bool {
	ng := lb.(*Nginx)
	filePath := n.GetConfigFilePath(ng.Protocol, ng.Name)
	needRelod, _ := n.Render(filePath, nginxTmpl, lb)
	return needRelod
}

func (n *NginxImpl) DeleteService(name string) bool {
	prot := []string{"HTTP", "L4"}

	for _, p := range prot {
		filename := n.GetConfigFilePath(p, name)
		if isExist, _ := utils.IsFileExist(filename); isExist {
			glog.Infof("Delete lb file %v", filename)
			if err := os.Remove(filename); err != nil {
				glog.Warningf("Failed to delete %v: %v", filename, err)
				return false
			}
			return true
		}
	}
	return false
}

func (n *NginxImpl) Reload() {
	//TODO rate limit
	check := fmt.Sprintf(checkPattern, cmd)
	if err := utils.ShellOut(check); err != nil {
		glog.Warningf("Invalid nginx configuration detected, error: %s", err)
		return
	}
	cmd := fmt.Sprintf(reloadPattern, cmd)
	if err := utils.ShellOut(cmd); err != nil {
		glog.Warningf("Reloading nginx failed, error: %s", err)
		return
	}
	glog.Infof("Reloading nginx")
}

func (n *NginxImpl) Recover() error {
	return nil
}

func (n *NginxImpl) Init() {
	if utils.CheckThreadExist(pidFile, cmd) {
		glog.Infof("nginx is already started")
		return
	}

	if err := utils.ShellOut(cmd); err != nil {
		glog.Fatalf("Failed to start nginx")
	}
}
