package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/golang/glog"
)

var (
	ipcamsMutex sync.Mutex
)

func InitAndRefreshIpcams() chan bool {
	registryOfflines()

	endRefreshIpcams := make(chan bool)
	go registryOfflinesPeriod(endRefreshIpcams)
	return endRefreshIpcams
}

func registryOfflines() {
	ipcamsMutex.Lock()
	defer ipcamsMutex.Unlock()
	for i, _ := range config.Ipcams {
		cam := &config.Ipcams[i]
		cam.Online = cam.Online || (!cam.Off && conductor.Registry(cam.Url))
	}
}

func registryOfflinesPeriod(end chan bool) {
	ticker := time.NewTicker(config.PingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-end:
			return
		case <-ticker.C:
			registryOfflines()
		}
	}
}

func OnGetIpcamsInfo(send chan []byte) {
	ipcamsMutex.Lock()
	defer ipcamsMutex.Unlock()

	// TODO add type wrap for server parsing
	info, err := json.Marshal(config.Ipcams)
	if err != nil {
		glog.Errorln(err)
		return
	}
	send <- info
}

func OnForceReRegistry(onlined string) {
	ipcamsMutex.Lock()
	defer ipcamsMutex.Unlock()
	for i, _ := range config.Ipcams {
		if cam := &config.Ipcams[i]; cam.Url == onlined {
			cam.Online = !cam.Off && conductor.Registry(cam.Url)
		}
	}
}
