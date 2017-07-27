package stats

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArikaChen/lbmanager/pkg/conf"

	"github.com/golang/glog"
)

const (
	errFileNotExist = "no such file or directory"
)

func Start() {
	go getCps()
}

type LvsStats struct {
	IP   string `json:"ip,omitempty"`
	Conn string `json:"conns,omitempty"`
}

func getStats(w http.ResponseWriter, r *http.Request) {
	var statsList []LvsStats
	data, err := ioutil.ReadFile(conf.Get().L4Conf.Stats.ResultPath)
	if err != nil {
		if !strings.Contains(err.Error(), errFileNotExist) {
			glog.Errorf("Failed to get stats, error: %s", err)
		}
	} else {
		ls := strings.Split(string(data), "\n")
		for _, v := range ls {
			ss := strings.Split(v, " ")
			if len(ss) != 12 {
				continue
			}
			if ss[0] == ss[1] || strings.Contains(ss[1], "127.0.0.1") {
				continue
			}
			statsList = append(statsList, LvsStats{IP: ss[1], Conn: ss[2]})
		}
	}
	if len(statsList) == 0 {
		return
	}

	wData, err := json.Marshal(statsList)
	if err != nil {
		glog.Errorf("Failed to marshal stats, error: %s", err)
		return
	}
	_, err = w.Write(wData)
	if err != nil {
		glog.Errorf("Failed to response stats, error: %s", err)
	}
}

func getCps() {
	for {
		http.HandleFunc("/stats", getStats)
		glog.Infof("Start lvs http server")
		err := http.ListenAndServe(":"+strconv.Itoa(conf.Get().L4Conf.Stats.Port), nil)
		if err != nil {
			glog.Errorf("Failed to start lvs http server, error: %s", err)
			time.Sleep(5 * time.Second)
		}
	}
}
