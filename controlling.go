package main

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Command struct {
	Name     string `json:"name"`
	Reciever string `json:"reciever"`
	Camera   string `json:"camera"`
}

func CtrlConnect() {
	connect(config.CtrlUrl(), onCtrlConnected)
}

func onCtrlConnected(ws *websocket.Conn) {
	send := make(chan []byte, 64)
	go writing(ws, send)
	OnGetIpcamsInfo(send)

	var command Command
	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		glog.Infoln("From one ctrl:", string(b))

		if err = json.Unmarshal(b, &command); err != nil {
			glog.Errorln(err)
			continue
		}

		switch command.Name {
		case "GetIpcamsInfo":
			OnGetIpcamsInfo(send)
		case "CreateSignalingConnection":
			go connectSignaling(config.SignalingUrl(command.Reciever), command.Camera, OnCreateSignalingConnection)
		case "ReconnectIpcam":
			OnReconnectIpcam(command.Camera)
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}
