package register

import (
	"encoding/json"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	. "github.com/empirefox/ic-client-one/center"
)

type RegMessage struct {
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

func ServeRegister(center *Center) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := center.Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		conn := NewConn(center, ws)
		defer close(conn.Send)

		center.AddStatusReciever <- conn
		defer func() { center.RemoveStatusReciever <- conn }()

		go conn.WriteClose()

		for {
			msg := &RegMessage{}
			if err := ws.ReadJSON(msg); err != nil {
				glog.Errorln(err)
				return
			}
			onRegister(center, conn, msg)
		}
	}
}

func onRegister(center *Center, conn *Connection, msg *RegMessage) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	switch msg.Type {
	case "GetStatus":
		status, err := center.GetStatus()
		if err != nil {
			status = []byte(`{"type":"Status","content":"error"}`)
		}
		conn.Send <- status
	case "GetRoomInfo":
		info, err := json.Marshal(gin.H{
			"type": "RoomInfo",
			"content": gin.H{
				"pid": syscall.Getpid(),
			},
		})
		if err != nil {
			glog.Errorln(err)
		}
		conn.Send <- info
	case "SetSecretAddress":
		center.OnSetSecretAddress([]byte(msg.Content))
	case "RemoveRoom":
		center.ChangeStatus <- "removing"
		center.OnRemoveRoom()
	case "Close":
		return
	case "Exit":
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	default:
		glog.Errorln("Unknow reg message", msg)
	}
}
