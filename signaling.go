package main

import (
	"encoding/json"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Signal struct {
	Type string `json:"type,omitempty"`

	Candidate string `json:"candidate,omitempty"`
	Mid       string `json:"sdpMid,omitempty"`
	Line      int    `json:"sdpMLineIndex,omitempty"`

	Sdp string `json:"sdp,omitempty"`
	Url string `json:"url,omitempty"`
}

func OnCreateSignalingConnection(ws *websocket.Conn) {
	var pc *rtc.PeerConn
	defer func() {
		if pc != nil {
			pc.Delete()
			close(pc.ToPeerChan)
		}
	}()

	var signal Signal
	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		if err = json.Unmarshal(b, &signal); err != nil {
			glog.Errorln(err)
			continue
		}

		switch signal.Type {
		case "offer":
			if pc != nil {
				glog.Errorln("Peer has created.")
				break
			}
			pc = conductor.CreatePeer(signal.Url)
			go writing(ws, pc.ToPeerChan)
			pc.CreateAnswer(signal.Sdp)
		case "candidate":
			pc.AddCandidate(signal.Candidate, signal.Mid, signal.Line)
		default:
			glog.Errorln("Unknow signal json:", string(b))
		}

	}
}
