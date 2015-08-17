package main

import (
	"bufio"
	"flag"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/empirefox/ic-client-one/center"
	"github.com/facebookgo/httpdown"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func main() {
	cpath := flag.String("cpath", "", "config file path")
	flag.Parse()
	c := center.NewCentral(*cpath)
	if err := c.Start(); err != nil {
		glog.Errorln(err)
		return
	}
	defer c.Close()

	router := gin.Default()
	router.GET("/local", c.ServeLocal)

	go readLineToQuit()

	server := &http.Server{Addr: ":12301", Handler: router}
	hd := &httpdown.HTTP{
		StopTimeout: 1 * time.Second,
		KillTimeout: 2 * time.Second,
	}
	if err := httpdown.ListenAndServe(server, hd); err != nil {
		glog.Errorln("httpdown.ListenAndServe: ", err)
	}
}

func readLineToQuit() {
	reader := bufio.NewReader(os.Stdin)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Errorln(err)
			}
		}()
		b, _, _ := reader.ReadLine()
		if string(b) == "exit" {
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			return
		}
	}()
}
