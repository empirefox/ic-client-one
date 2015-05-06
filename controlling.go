package main

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Command struct {
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

func CtrlConnect() {
	connect(config.CtrlUrl(), onCtrlConnected)
}

func onCtrlConnected(ws *websocket.Conn) {
	send := make(chan []byte, 64)
	go writing(ws, send)

	var command Command
	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		if err = json.Unmarshal(b, &command); err != nil {
			glog.Errorln(err)
			continue
		}

		switch command.Name {
		case "GetIpcamsInfo":
			OnGetIpcamsInfo(send)
		case "CreateSignalingConnection":
			connect(config.SignalingUrl(), OnCreateSignalingConnection)
		case "ForceReRegistry":
			OnForceReRegistry(command.Url)
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}
