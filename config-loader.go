package main

import (
	"regexp"
	"time"

	"github.com/dchest/uniuri"
)

var config Config
var whitespaceRegexp = regexp.MustCompile("\\s")

func init() {
	config = ParseConfig()
}

func ParseConfig() Config {
	c := Config{
		Secure:     false,
		Server:     "127.0.0.1:9999",
		PingPeriod: 45 * time.Second,
		Ipcams: []ConfigIpcam{
			{
				Id:   uniuri.NewLen(32),
				Name: "客厅",
				Url:  "rtsp://127.0.0.1:1235/test1.sdp",
			},
		},
	}
	return c
}
