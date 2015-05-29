package main

import (
	"flag"
	"net/http"

	"github.com/empirefox/ic-client-one/center"
	"github.com/empirefox/ic-client-one/controlling"
	"github.com/empirefox/ic-client-one/register"
	"github.com/fvbock/endless"
	"github.com/golang/glog"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

func main() {
	flag.Parse()
	c := center.NewCenter()
	c.Start()
	defer c.Close()

	go controlling.CtrlConnect(c)

	http.HandleFunc("/register", register.ServeRegister(c))
	err := endless.ListenAndServe(":12301", nil)
	if err != nil {
		glog.Fatalln("ListenAndServe: ", err)
	}
}
