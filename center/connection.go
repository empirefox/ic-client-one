package center

import (
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

// implements Ws
type connection struct {
	*websocket.Conn
	send    chan []byte
	central Central
	quit    chan struct{}
}

func NewConn(central Central, ws *websocket.Conn, quit chan struct{}) Ws {
	return &connection{
		Conn:    ws,
		send:    make(chan []byte, 64),
		central: central,
		quit:    quit,
	}
}

func (conn connection) Send(msg []byte) {
	conn.send <- msg
}

func (conn connection) WriteClose() {
	ticker := time.NewTicker(conn.central.Conf().GetPingPeriod())
	defer func() {
		glog.Infoln("conn closing")
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case msg, ok := <-conn.send:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				glog.Infoln("conn send closing")
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				glog.Infoln("conn send error:", string(msg), err)
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				glog.Infoln("conn send ping error", err)
				return
			}
		case <-conn.quit:
			return
		}
	}
	return
}
