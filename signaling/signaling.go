package signaling

import (
	"encoding/json"

	"github.com/golang/glog"

	"github.com/empirefox/ic-client-one-wrap"
	. "github.com/empirefox/ic-client-one/center"
	. "github.com/empirefox/ic-client-one/utils"
)

type Signal struct {
	Type string `json:"type,omitempty"`

	Candidate string `json:"candidate,omitempty"`
	Mid       string `json:"sdpMid,omitempty"`
	Line      int    `json:"sdpMLineIndex,omitempty"`

	Sdp string `json:"sdp,omitempty"`
}

type SubSignalCommand struct {
	Camera   string `json:"camera,omitempty"`
	Reciever string `json:"reciever,omitempty"`
}

// Content => SubSignalCommand
func OnCreateSignalingConnection(center *Center, cmd *Command) {
	var sub SubSignalCommand
	if err := json.Unmarshal([]byte(cmd.Content), &sub); err != nil {
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Cannot parse SubSignalCommand")
		return
	}
	glog.Infoln("connect to", sub.Camera)
	ws, _, err := center.Dialer.Dial(center.Conf.SignalingUrl(sub.Reciever), nil)
	if err != nil {
		glog.Errorln(err)
		center.CtrlConn.Send <- GenInfoMessage(cmd.From, "Dial signaling failed")
		return
	}
	//	defer ws.Close()
	conn := NewConn(center, ws)
	go conn.WriteClose()
	onSignalingConnected(conn, center.Conf.GetIpcamUrl(sub.Camera))
}

func onSignalingConnected(conn *Connection, url string) {
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
				var ok = false
				pc, ok = conn.Center.CreatePeer(url, conn)
				if !ok {
					pc = nil
					return
				}
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
