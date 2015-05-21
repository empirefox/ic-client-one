package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/empirefox/ic-client-one-wrap"
	"github.com/fvbock/endless"
	"github.com/golang/glog"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

var (
	conductor rtc.Conductor
)

func InitConductor() {
	conductor = rtc.NewShared()
	conductor.AddIceUri("stun:stun.l.google.com:19302")
	conductor.AddIceUri("stun:stun.anyfirewall.com:3478")
	conductor.AddIceServer("turn:turn.bistri.com:80", "homeo", "homeo")
	conductor.AddIceServer("turn:turn.anyfirewall.com:443?transport=tcp", "webrtc", "webrtc")
}

func main() {
	flag.Parse()
	center := NewCenter()
	InitConductor()
	go InitAndRefreshIpcams(center.Quit)
	go CtrlConnect(center)

	http.HandleFunc("/register", serveRegister(center))
	err := endless.ListenAndServe(":12301", nil)
	if err != nil {
		glog.Fatalln("ListenAndServe: ", err)
	}

	center.Close()
	conductor.Release()
	os.Exit(0)
}
