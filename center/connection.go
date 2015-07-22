package center

import (
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
}

type Dialer interface {
	Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

type Ws interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	ReadJSON(v interface{}) error
	Close() error
}

type Connection struct {
	Ws
	Send   chan []byte
	Center *Center
}

func NewConn(center *Center, ws *websocket.Conn) *Connection {
	return &Connection{
		Ws:     ws,
		Send:   make(chan []byte, 64),
		Center: center,
	}
}

func (ws Connection) WriteClose() (quitLoop bool) {
	ticker := time.NewTicker(ws.Center.Conf.PingPeriod)
	defer func() {
		glog.Infoln("ws closing")
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
		ticker.Stop()
		if !quitLoop {
			ws.Close()
		}
	}()
	for {
		select {
		case msg, ok := <-ws.Send:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, []byte{})
				glog.Infoln("ws send closing")
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				glog.Infoln("ws send error:", string(msg), err)
				return
			}
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				glog.Infoln("ws send ping error", err)
				return
			}
		case <-ws.Center.Quit:
			quitLoop = true
			return
		}
	}
	return
}
