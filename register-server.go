package main

import (
	"net/http"

	"github.com/golang/glog"
)

type RegMessage struct {
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

func serveRegister(center *Center) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		ws, err := center.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()

		conn := NewConn(center, ws)
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
			default:
				glog.Errorln("Unknow reg message")
			}
		}
	}
}
