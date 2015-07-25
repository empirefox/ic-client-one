package controlling

import (
	"time"

	"github.com/golang/glog"

	. "github.com/empirefox/ic-client-one/center"
	"github.com/empirefox/ic-client-one/signaling"
	"github.com/empirefox/ic-client-one/wsio"
)

func CtrlConnect(center *Center) {
	for !ctrlConnectLoop(center) {
	}
}

func ctrlConnectLoop(center *Center) (quitLoop bool) {
	ws, _, err := center.Dialer.Dial(center.Conf.CtrlUrl(), nil)
	if err != nil {
		glog.Errorln(err)
		center.ChangeStatus <- "unreachable"
		time.Sleep(time.Second * 10)
		return
	}

	conn := NewConn(center, ws)
	center.AddCtrlConn(conn)
	defer center.RemoveCtrlConn()

	go onCtrlConnected(conn)
	if quitLoop = conn.WriteClose(); !quitLoop {
		time.Sleep(time.Second * 10)
	}
	return
}

func onCtrlConnected(c *Connection) {
	defer c.Close()
	addr := c.Center.Conf.GetAddr()
	if len(addr) == 0 {
		c.Center.ChangeStatus <- "not_authed"
		return
	}
	c.Center.ChangeStatus <- "authing"
	// login
	c.Send <- append([]byte("addr:"), addr...)

	for {
		var cmd wsio.FromServerCommand
		if err := c.ReadJSON(&cmd); err != nil {
			glog.Errorln(err)
			c.Center.ChangeStatus <- "not_ready"
			return
		}

		switch cmd.Name {
		case "GetIpcams":
			c.Center.OnGetIpcams()
		case "ManageGetIpcam":
			c.Center.OnManageGetIpcam(&cmd)
		case "ManageSetIpcam":
			c.Center.OnManageSetIpcam(&cmd)
		case "ManageReconnectIpcam":
			c.Center.OnManageReconnectIpcam(&cmd)
		case "CreateSignalingConnection":
			go signaling.OnCreateSignalingConnection(c.Center, &cmd)
		case "LoginAddrOk":
			c.Center.OnGetIpcams()
			c.Center.ChangeStatus <- "ready"
		case "LoginAddrError":
			glog.Infoln("auth_failed")
			c.Center.ChangeStatus <- "auth_failed"
			return
		default:
			glog.Errorln("Unknow command json", cmd)
		}
	}
	c.Center.ChangeStatus <- "not_ready"
}
