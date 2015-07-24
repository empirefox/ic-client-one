package main

import (
	"bufio"
	"flag"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/empirefox/ic-client-one/center"
	"github.com/empirefox/ic-client-one/controlling"
	"github.com/empirefox/ic-client-one/register"
	"github.com/facebookgo/httpdown"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

func main() {
	cpath := flag.String("cpath", "", "config file path")
	flag.Parse()
	c := center.NewCenter(*cpath)
	if err := c.Start(); err != nil {
	}
	defer func() {
		c.Close()
		os.Exit(0)
	}()

	go controlling.CtrlConnect(c)

	router := gin.Default()
	router.GET("/register", register.ServeRegister(c))

	go readLineToQuit()

	server := &http.Server{Addr: ":12301", Handler: router}
	hd := &httpdown.HTTP{
		StopTimeout: 1 * time.Second,
		KillTimeout: 1 * time.Second,
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
