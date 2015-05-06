package main

import (
	"fmt"
	"time"
)

type ConfigIpcam struct {
	Name   string `json:"name,omitempty"`
	Url    string `json:"url,omitempty"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"`
}

type Config struct {
	Secure     bool
	Server     string // ip:port
	PingPeriod time.Duration
	Ipcams     []ConfigIpcam
}

func (c *Config) wsUrl(context string) string {
	p := "ws"
	if c.Secure {
		p = "wss"
	}
	return fmt.Sprintf("%s://%s/one/%s", p, c.Server, context)
}

func (c *Config) CtrlUrl() string {
	return c.wsUrl("ctrl")
}

func (c *Config) SignalingUrl() string {
	return c.wsUrl("signaling")
}
