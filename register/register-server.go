package register

import (
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
			var msg RegMessage
			if err := ws.ReadJSON(&msg); err != nil {
				glog.Errorln(err)
				return
			}

			switch msg.Type {
			case "GetStatus":
				status, err := center.GetStatus()
				if err != nil {
					status = []byte(`{"type":"Status","content":"error"}`)
				}
				conn.Send <- status
			case "SetSecretAddress":
				center.OnSetSecretAddress(msg.Content)
			case "Exit":
				syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			default:
				glog.Errorln("Unknow reg message")
			}
		}
	}
}
