package main

import (
	"flag"

	"github.com/empirefox/ic-client-one-wrap"
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
	InitConductor()
	endRefreshIpcams := InitAndRefreshIpcams()

	CtrlConnect()

	endRefreshIpcams <- true
	conductor.Release()
	glog.Infoln("quit now")
}
