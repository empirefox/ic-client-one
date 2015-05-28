package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dchest/uniuri"
	"github.com/golang/glog"
)

var (
	configFile = "./config.toml"
)

type Ipcam struct {
	Id     string `json:"id,omitempty"      toml:"-"`
	Name   string `json:"name,omitempty"`
	Url    string `json:"-"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"`
}

type Config struct {
	file          string `toml:"-"`
	SecretAddress string
	Secure        bool
	Server        string // ip:port
	PingPeriod    time.Duration
	Ipcams        []Ipcam
}

func NewConfig() *Config {
	return newConfig(configFile)
}

func newConfig(confFile string) *Config {
	conf := &Config{file: confFile}
	if err := conf.Load(); err != nil {
		panic(err)
	}
	return conf
}

func (c *Config) Load() error {
	if _, err := os.Stat(c.file); os.IsNotExist(err) {
		glog.Errorln("no such file or directory: %s", c.file)
		return nil
	}
	if _, err := toml.DecodeFile(c.file, c); err != nil {
		glog.Errorln(err)
		return err
	}
	for i := range c.Ipcams {
		c.Ipcams[i].Id = uniuri.NewLen(16)
	}
	return nil
}

func (c *Config) Save() error {
	file, err := os.Create(c.file)
	if err != nil {
		glog.Errorln(err)
		return err
	}
	err = toml.NewEncoder(file).Encode(c)
	if err != nil {
		glog.Errorln(err)
	}
	return nil
}

func (c *Config) SetAddr(addr string) error {
	c.SecretAddress = addr
	return c.Save()
}

func (c *Config) SaveIpcam(updated Ipcam) error {
	if updated.Id == "" {
		updated.Id = uniuri.NewLen(16)
		c.Ipcams = append(c.Ipcams, updated)
		return c.Save()
	}
	for i, ipcam := range c.Ipcams {
		if updated.Id == ipcam.Id {
			c.Ipcams[i] = updated
			return c.Save()
		}
	}
	return errors.New("Wrong ipcam to save")
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
