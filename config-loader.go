package main

import "time"

var config Config

func init() {
	config = ParseConfig()
}

func ParseConfig() Config {
	return Config{
		Secure:     false,
		Server:     "192.168.1.222:9999",
		PingPeriod: 45 * time.Second,
		Ipcams: []ConfigIpcam{
			{
				Name: "客厅",
				Url:  "rtsp://127.0.0.1:1235/test1.sdp",
			},
		},
	}
}
