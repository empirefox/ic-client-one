package main

import (
	"crypto/tls"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

var dailer = websocket.Dialer{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

func connect(url string, handler func(*websocket.Conn)) {
	glog.Infoln("connect to", url)
	ws, _, err := dailer.Dial(url, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	glog.Infoln("connected ws to", url)
	handler(ws)
}

func connectSignaling(url, id string, handler func(*websocket.Conn, string)) {
	glog.Infoln("connect to", url)
	ipcamUrl := ""
	for _, ipcam := range config.Ipcams {
		if ipcam.Id == id {
			ipcamUrl = ipcam.Url
			break
		}
	}
	if ipcamUrl == "" {
		glog.Errorln("Cannot find ipcam url")
		return
	}
	ws, _, err := dailer.Dial(url, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	glog.Infoln("connected ws to", ipcamUrl)
	handler(ws, ipcamUrl)
}

func writing(ws *websocket.Conn, send chan []byte) {
	ticker := time.NewTicker(config.PingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()
	for {
		select {
		case msg, ok := <-send:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
			glog.Infoln("ws send ", string(msg))
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
