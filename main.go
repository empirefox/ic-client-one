package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

var dailer = websocket.Dialer{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

type PeerMsg struct {
	Candidate string `json:"candidate,omitempty"`
	Mid       string `json:"sdpMid,omitempty"`
	Line      int    `json:"sdpMLineIndex,omitempty"`

	Type string `json:"type,omitempty"`
	Sdp  string `json:"sdp,omitempty"`
}

func readMsgs(ws *websocket.Conn, pc *rtc.PeerConn) {
	defer close(pc.ToPeerChan)

	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		var msg PeerMsg
		if json.Unmarshal(b, &msg) == nil {
			switch msg.Type {
			case "offer":
				// offer comes
				glog.Infoln("offer comes after running")
				*pc = rtc.CreatePeer()
				pc.CreateAnswer(msg.Sdp)
			case "candidate":
				// cadidate comes
				pc.AddCandidate(msg.Candidate, msg.Mid, msg.Line)
			default:
				glog.Errorln("got unknow json message:", string(b))
			}
		}
	}
}

func startWs() {
	ws, _, err := dailer.Dial("ws://192.168.1.222:9999/one", nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()

	glog.Infoln("ws connected")
	_, b, err := ws.ReadMessage()
	if err != nil {
		glog.Errorln(err)
		return
	}
	var offer PeerMsg
	if json.Unmarshal(b, &offer) != nil || offer.Type != "offer" {
		glog.Errorln("must be offer, but:", offer)
		return
	}
	// offer comes
	pc := rtc.CreatePeer()
	addICE(&pc)
	pc.CreateAnswer(offer.Sdp)
	glog.Infoln("CreateAnswer ok")
	go readMsgs(ws, &pc)

	for {
		select {
		case msg := <-pc.ToPeerChan:
			if msg == "" {
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		}
	}
}

func addICE(pc *rtc.PeerConn) {
	pc.AddIceUri("stun:stun.l.google.com:19302")
	pc.AddIceUri("stun:stun.anyfirewall.com:3478")
	pc.AddIceServer("turn:turn.bistri.com:80", "homeo", "homeo")
	pc.AddIceServer("turn:turn.anyfirewall.com:443?transport=tcp", "webrtc", "webrtc")
}

func main() {
	flag.Parse()
	startWs()
}
