package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/golang/glog"
)

type ConfigIpcam struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Url    string `json:"-"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"`
}

type Config struct {
	SecretAddress string
	Secure        bool
	Server        string // ip:port
	PingPeriod    time.Duration
	Ipcams        []ConfigIpcam
}

func (c *Config) Save() error {
	file, err := os.Create(configFile)
	if err != nil {
		glog.Errorln(err)
		return err
	}
	err = toml.NewEncoder(file).Encode(c)
	if err != nil {
		glog.Errorln(err)
	}
	return err
}

func (c *Config) SetSecretAddress(addr string) error {
	c.SecretAddress = addr
	return c.Save()
}

func (c *Config) GetIpcamUrl(id string) string {
	for _, ipcam := range c.Ipcams {
		if ipcam.Id == id {
			return ipcam.Url
		}
	}
	glog.Errorln("Cannot find ipcam url")
	return ""
}

func (c *Config) GetOrigin() string {
	return strings.Split(c.Server, ":")[0]
}

func (c *Config) GetAddr() []byte {
	return []byte(c.SecretAddress)
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

func (c *Config) SignalingUrl(reciever string) string {
	return c.wsUrl("signaling/" + reciever)
}
