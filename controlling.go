package main

import (
	"time"

	"github.com/golang/glog"
)

type Command struct {
	Name     string `json:"name"`
	Reciever string `json:"reciever"`
	Camera   string `json:"camera"`
}

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
	defer ws.Close()

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
	addr := c.Center.Conf.GetAddr()
	if len(addr) == 0 {
		c.Center.ChangeStatus <- "not_authed"
		return
	}
	c.Center.ChangeStatus <- "ready"
	defer func() { c.Center.ChangeStatus <- "not_ready" }()
	// login
	c.Send <- addr
	c.Center.OnGetIpcamsInfo(c.Send)

	for {
		var command Command
		if err := c.ReadJSON(&command); err != nil {
			return
		}

		switch command.Name {
		case "GetIpcamsInfo":
			c.Center.OnGetIpcamsInfo(c.Send)
		case "SaveIpcam":
			c.Center.OnSaveIpcam(command, c.Send)
		case "CreateSignalingConnection":
			go OnCreateSignalingConnection(c.Center, &command)
		case "ReconnectIpcam":
			c.Center.OnReconnectIpcam(command.Camera)
		default:
			glog.Errorln("Unknow command json")
		}
	}
}
