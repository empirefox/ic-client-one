package main

import (
	"github.com/empirefox/ic-client-one-wrap"
	"github.com/golang/glog"
)

type Signal struct {
	Type string `json:"type,omitempty"`

	Candidate string `json:"candidate,omitempty"`
	Mid       string `json:"sdpMid,omitempty"`
	Line      int    `json:"sdpMLineIndex,omitempty"`

	Sdp string `json:"sdp,omitempty"`
}

func OnCreateSignalingConnection(center *Center, cmd *Command) {
	glog.Infoln("connect to", cmd.Camera)
	ws, _, err := dailer.Dial(config.SignalingUrl(cmd.Reciever), nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	glog.Infoln("connected")
	conn := NewConn(center, ws)
	go conn.WriteClose()
	onSignalingConnected(conn, config.GetIpcamUrl(cmd.Camera))
}

func onSignalingConnected(conn *Connection, url string) {
	var pc *rtc.PeerConn
	defer func() {
		if pc != nil {
			pc.Delete()
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
				pc = conductor.CreatePeer(url, conn.Send)
				pc.CreateAnswer(signal.Sdp)
			}
		case "candidate":
			if pc == nil {
				return
			}
			pc.AddCandidate(signal.Candidate, signal.Mid, signal.Line)
		default:
			glog.Errorln("Unknow signal json")
		}

	}
}
