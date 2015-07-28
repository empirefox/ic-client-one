package signaling

import (
	"github.com/golang/glog"

	"github.com/empirefox/ic-client-one-wrap"
	. "github.com/empirefox/ic-client-one/center"
	"github.com/empirefox/ic-client-one/ipcam"
	. "github.com/empirefox/ic-client-one/utils"
	"github.com/empirefox/ic-client-one/wsio"
)

type Signal struct {
	Type string `json:"type,omitempty"`

	Candidate string `json:"candidate,omitempty"`
	Mid       string `json:"sdpMid,omitempty"`
	Line      int    `json:"sdpMLineIndex,omitempty"`

	Sdp string `json:"sdp,omitempty"`
}

// Camera => Id
// Content => SubSignalCommand
// cmd from signaling-server many.go CreateSignalingConnectionCommand
func OnCreateSignalingConnection(center *Center, cmd *wsio.FromServerCommand) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()

	sub, err := cmd.Signaling()
	if err != nil {
		glog.Errorln(*cmd)
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot parse SubSignalCommand")
		return
	}
	i, err := center.Conf.GetIpcam([]byte(sub.Camera))
	if err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Camera not found")
		return
	}
	ws, _, err := center.Dialer.Dial(center.Conf.SignalingUrl(sub.Reciever), nil)
	if err != nil {
		glog.Errorln(err)
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Dial signaling failed")
		return
	}
	//	defer ws.Close()
	conn := NewConn(center, ws)
	go conn.WriteClose()
	onSignalingConnected(conn, i)
}

func onSignalingConnected(conn *Connection, i ipcam.Ipcam) {
	var pc rtc.PeerConn
	defer func() {
		glog.Infoln("onSignalingConnected finished")
		if pc != nil {
			glog.Infoln("deleting peer")
			conn.Center.Conductor.DeletePeer(pc)
		}
	}()

	for {
		var signal Signal
		if err := conn.ReadJSON(&signal); err != nil {
			glog.Errorln(err)
			return
		}

		switch signal.Type {
		case "offer":
			if pc == nil {
				glog.Infoln("creating peer")
				if !i.Online {
					return
				}
				pc = conn.Center.Conductor.CreatePeer(i.Url, conn.Send)
				pc.CreateAnswer(signal.Sdp)
			}
		case "candidate":
			if pc != nil {
				glog.Infoln("add candidate")
				pc.AddCandidate(signal.Candidate, signal.Mid, signal.Line)
			}
		default:
			glog.Errorln("Unknow signal json")
		}

	}
}
