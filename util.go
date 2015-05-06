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
	ws, _, err := dailer.Dial(url, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	handler(ws)
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
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
