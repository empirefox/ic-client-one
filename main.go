package main

import (
	"flag"
	"net/http"

	"github.com/fvbock/endless"
	"github.com/golang/glog"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

func main() {
	flag.Parse()
	center := NewCenter()
	center.Start()
	go CtrlConnect(center)

	http.HandleFunc("/register", serveRegister(center))
	err := endless.ListenAndServe(":12301", nil)
	if err != nil {
		glog.Fatalln("ListenAndServe: ", err)
	}

	center.Close()
}
