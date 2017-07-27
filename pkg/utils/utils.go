package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

func ShellOut(cmd string) (err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	glog.V(3).Infof("Executing %s", cmd)

	command := exec.Command("sh", "-c", cmd)
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Start()
	if err != nil {
		return fmt.Errorf("Failed to execute %v, err: %v", cmd, err)
	}

	err = command.Wait()
	if err != nil {
		return fmt.Errorf("Command %v stdout: %q\nstderr: %q\nfinished with error: %v", cmd,
			stdout.String(), stderr.String(), err)
	}
	return nil
}

func IsFileExist(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}

func getPidByFile(pidFile string) (string, error) {
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		glog.V(3).Infof("Failed to read pid file %s", pidFile)
		return "", err
	}
	str := string(data[:])
	str = strings.Replace(str, "\n", "", -1)
	return str, nil
}

func GetPid(pidFile string, cmd string) (string, error) {
	pid, err := getPidByFile(pidFile)
	if err != nil {
		return "", err
	}
	proc := fmt.Sprintf("/proc/%s/exe", pid)
	_, err = IsFileExist(proc)
	if err != nil {
		glog.V(3).Infof("Failed to get proc file, pid: %d", pid)
		return "", err
	}
	exe, err := os.Readlink(proc)
	if err != nil {
		glog.V(3).Infof("Failed to get exe file, pid: %d", pid)
		return "", err
	}
	if exe != cmd {
		glog.V(3).Infof("Is not the same thread, pid: %d, old: %s, new: %s", pid, exe, cmd)
		return "", errors.New("Thread exec is wrong")
	}
	return pid, nil
}

func CheckThreadExist(pidFile string, cmd string) bool {
	pid, _ := GetPid(pidFile, cmd)
	return pid != ""
}

func IsElementExist(s string, l []string) bool {
	for _, e := range l {
		if s == e {
			return true
		}
	}
	return false
}
