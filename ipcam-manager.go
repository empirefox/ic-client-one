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

	ipcams := make(map[string]ConfigIpcam, len(config.Ipcams))
	for _, ipcam := range config.Ipcams {
		ipcams[ipcam.Id] = ipcam
	}
	info, err := json.Marshal(ipcams)
	if err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("send IpcamsInfo")
	// Give a type header
	send <- append([]byte("one:IpcamsInfo:"), info...)
}

func OnReconnectIpcam(id string) {
	ipcamsMutex.Lock()
	defer ipcamsMutex.Unlock()
	for i, _ := range config.Ipcams {
		if cam := &config.Ipcams[i]; cam.Id == id {
			cam.Online = !cam.Off && conductor.Registry(cam.Url)
		}
	}
}
