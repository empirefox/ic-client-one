package config

import (
	"errors"
	"fmt"
	"os"
	"path"
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
	Rec    bool   `json:"-"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"  toml:"-"`
}

func (i *Ipcam) GetRecName(conf *Config) string {
	return path.Join(conf.RecDir, i.Name)
}

type Config struct {
	file          string `toml:"-"`
	RecDir        string
	SecretAddress string
	Secure        bool
	Server        string // ip:port
	PingPeriod    time.Duration
	Ipcams        []Ipcam
}

func NewConfig() *Config {
	return newConfig(configFile)
}

func NewConfigFile(confFile string) *Config {
	return newConfig(confFile)
}

func newConfig(confFile string) *Config {
	conf := &Config{file: confFile}
	if err := conf.Load(); err != nil {
		panic(err)
	}
	err := os.MkdirAll(conf.RecDir, os.ModePerm)
	if err != nil {
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
	for i := range c.Ipcams {
		if updated.Id == c.Ipcams[i].Id {
			c.Ipcams[i] = updated
			return c.Save()
		}
	}
	return errors.New("Wrong ipcam to save")
}

func (c *Config) GetIpcam(id string) (*Ipcam, error) {
	for i := range c.Ipcams {
		if c.Ipcams[i].Id == id {
			return &c.Ipcams[i], nil
		}
	}
	return nil, errors.New("Cannot find ipcam")
}

func (c *Config) GetIpcamUrl(id string) string {
	for i := range c.Ipcams {
		if c.Ipcams[i].Id == id {
			return c.Ipcams[i].Url
		}
	}
	glog.Errorln("Cannot find ipcam url")
	return ""
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

func (c *Config) RegRoomUrl() string {
	p := "http"
	if c.Secure {
		p = "https"
	}
	return fmt.Sprintf("%s://%s/%s", p, c.Server, "login.html?from=/regroom.html")
}
