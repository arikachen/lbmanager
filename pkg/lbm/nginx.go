package lbm

import (
	"encoding/json"
	"path"

	"github.com/golang/glog"
)

const (
	checkPattern  = "%s -t"
	reloadPattern = "%s -s reload"
	pidFile       = "/run/nginx.pid"
)

type NginxImpl struct {
	LBBase

	streamConfPath string
	nginxCMD       string
	local          bool
}

func NewNginxImpl() *NginxImpl {
	lm := &NginxImpl{
		LBBase:   NewLB(),
		nginxCMD: "",
	}
	lm.ConfigPath = path.Join(lm.Conf.ConfDir, "conf.d")
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

func (n *NginxImpl) ConfigService(lb interface{}) bool {
	ng := lb.(*Nginx)
	filePath := n.GetConfigFilePath(ng.Name)
	needRelod, _ := n.Render(filePath, nginxTmpl, lb)
	return needRelod
}

func (n *NginxImpl) DeleteService(name string) bool {
	return false
}

func (n *NginxImpl) Reload() {
	/*
		check := fmt.Sprintf(checkPattern, n.nginxCMD)
		if err := utils.ShellOut(check); err != nil {
			return fmt.Errorf("Invalid nginx configuration detected, not reloading: %s", err)
		}
		cmd := fmt.Sprintf(reloadPattern, n.nginxCMD)
		if err := utils.ShellOut(cmd); err != nil {
			return fmt.Errorf("Reloading NGINX failed: %s", err)
		}*/
	glog.V(3).Info("Reloading nginx")
}

func (n *NginxImpl) Recover() error {
	return nil
}
