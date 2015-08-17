package center

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/empirefox/ic-client-one/storage"
	"github.com/empirefox/ic-client-one/wsio"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
}

type Dialer interface {
	Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

type Central interface {
	Upgrader
	Dialer
	ChangeStatus(status []byte)
	AddStatusObserver(ws Ws)
	DelStatusObserver(ws Ws)
	Conf() *storage.Conf
	SetCtrl(ws Ws)
	DelCtrl(ws Ws)
	SendCtrl(msg []byte)

	OnServerCommand(cmd *wsio.FromServerCommand)
	OnLocalCommand(cmd *FromLocalCommand)

	Start() error
	Close()
	ServeLocal(c *gin.Context)
}

type Socket interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	ReadJSON(v interface{}) error
	Close() error
}

type Ws interface {
	Socket
	Send(msg []byte)
	WriteClose()
}

// From Local
type FromLocalCommand struct {
	Ws      Ws              `json"-"`
	Type    string          `json:"type,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
}

func (c *FromLocalCommand) Value() []byte {
	return bytes.Trim(c.Content, `"`)
}
