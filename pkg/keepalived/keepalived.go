package keepalived

import (
	"fmt"
	"sync"

	"github.com/ArikaChen/lbmanager/pkg/conf"
	"github.com/ArikaChen/lbmanager/pkg/utils"

	"time"

	"github.com/golang/glog"
)

const (
	pidFile = "/var/run/keepalived.pid"
	cmd     = "/usr/sbin/keepalived"
	service = "keepalived"
)

var (
	toReload = false
	lock     sync.Mutex
)

func Start() {
	defer start()

	if utils.CheckThreadExist(pidFile, cmd) {
		glog.Infof("Keepalived is already started")
		return
	}
	scmd := fmt.Sprintf("systemctl start %s", service)
	if err := utils.ShellOut(scmd); err != nil {
		glog.Fatalf("Failed to start keepalived")
	}
}

func Reload() {
	lock.Lock()
	defer lock.Unlock()

	toReload = true
}

func start() {
	glog.Infof("Start keepalived reload thread")
	go reloadPeriod(cmd)
}

func reload(cmd string) error {
	glog.Infof("Reloading keepalived")
	pid, err := utils.GetPid(pidFile, cmd)
	if err == nil {
		cmd := fmt.Sprintf("/bin/kill -HUP %s", pid)
		err = utils.ShellOut(cmd)
		if err == nil {
			return nil
		}
	}
	glog.Errorf("Reloading keepalived failed: %s", err)
	return err
}

func reloadPeriod(cmd string) {
	for {
		r := false
		lock.Lock()
		r = toReload
		if toReload {
			err := reload(cmd)
			if err == nil {
				toReload = false
			}
		}
		lock.Unlock()

		if r {
			times := time.Duration(conf.Get().L4Conf.KConf.ReloadTime)
			time.Sleep(times * time.Second)
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}
