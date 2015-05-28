package main

import (
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Connection struct {
	*websocket.Conn
	Send   chan []byte
	Center *Center
}

func NewConn(center *Center, ws *websocket.Conn) *Connection {
	return &Connection{
		Conn:   ws,
		Send:   make(chan []byte, 64),
		Center: center,
	}
}

func (ws Connection) WriteClose() (quitLoop bool) {
	ticker := time.NewTicker(ws.Center.Conf.PingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()
	for {
		select {
		case msg, ok := <-ws.Send:
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
		case <-ws.Center.Quit:
			quitLoop = true
			return
		}
	}
	return
}
