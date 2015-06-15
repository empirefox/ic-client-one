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
