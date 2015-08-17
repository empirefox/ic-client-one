package center

import (
	"github.com/golang/glog"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/empirefox/ic-client-one/ipcam"
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
func (center *central) OnCreateSignalingConnection(cmd *wsio.FromServerCommand) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()

	sub, err := cmd.Signaling()
	if err != nil {
		glog.Errorln(*cmd)
		center.SendCtrl(cmd.ToManyInfo("Cannot parse SubSignalCommand"))
		return
	}
	i, err := center.conf.GetIpcam([]byte(sub.Camera))
	if err != nil {
		center.SendCtrl(cmd.ToManyInfo("Camera not found"))
		return
	}
	socket, _, err := center.Dial(center.conf.SignalingUrl(sub.Reciever), nil)
	if err != nil {
		glog.Errorln(err)
		center.SendCtrl(cmd.ToManyInfo("Dial signaling failed"))
		return
	}
	defer socket.Close()
	ws := NewConn(center, socket, center.quit)
	go ws.WriteClose()
	center.onSignalingConnected(ws, i)
}

func (center *central) onSignalingConnected(ws Ws, i ipcam.Ipcam) {
	var pc rtc.PeerConn
	defer func() {
		glog.Infoln("onSignalingConnected finished")
		if pc != nil {
			glog.Infoln("deleting peer")
			center.Conductor.DeletePeer(pc)
		}
	}()

	for {
		var signal Signal
		if err := ws.ReadJSON(&signal); err != nil {
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
				pc = center.Conductor.CreatePeer(i.Url, ws.Send)
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
