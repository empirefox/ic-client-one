package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	. "github.com/empirefox/ic-client-one/ipcam"
	"github.com/golang/glog"
)

const (
	FILE_MODE os.FileMode = 0644
)

var (
	ErrSystemBucketNotFound = errors.New("system bucket not found")
	ErrSystemKeyNotFound    = errors.New("system key not found")
	ErrIpcamNotFound        = errors.New("ipcam not found")
	ErrDbPathRequired       = errors.New("DbPath must be set")
	ErrRecDirRequired       = errors.New("RecDir must be set")
	ErrWsUrlRequired        = errors.New("WsUrl must be set")
	ErrPingSecond           = errors.New("PingSecond must greater than 30")
	ErrSetupParam           = errors.New("setup is not a valid json file nor valid json content")
	ErrEmptySetupParam      = errors.New("setup is empty")

	sysBucketName    = []byte("system")
	ipcamsBucketName = []byte("ipcams")

	K_REG_TOKEN  = []byte("RegToken")
	K_ROOM_TOKEN = []byte("RoomToken")
)

type Setup struct {
	DbPath     string
	RecDir     string
	WsUrl      string
	PingSecond time.Duration
	Stuns      []string
}

func (setup *Setup) Validate() error {
	if setup.WsUrl == "" {
		return ErrWsUrlRequired
	}
	if setup.PingSecond < 30 {
		return ErrPingSecond
	}
	if setup.DbPath == "" {
		return ErrDbPathRequired
	}
	if setup.RecDir == "" {
		return ErrRecDirRequired
	}
	setup.DbPath = os.ExpandEnv(setup.DbPath)
	setup.RecDir = os.ExpandEnv(setup.RecDir)
	return nil
}

///////////////////////////////////////////
// Conf
///////////////////////////////////////////
type Conf struct {
	setup Setup
	db    *bolt.DB
}

func NewConf(str string) (*Conf, error) {
	content, err1 := ioutil.ReadFile(str)
	if err1 != nil {
		content = []byte(str)
	}
	if len(content) == 0 {
		return nil, ErrEmptySetupParam
	}
	var setup Setup
	err2 := json.Unmarshal(content, &setup)
	if err2 != nil {
		if err1 != nil {
			return nil, ErrSetupParam
		}
		return nil, err2
	}
	if err := setup.Validate(); err != nil {
		return nil, err
	}
	return &Conf{setup: setup}, nil
}

func (c *Conf) Open() (err error) {
	c.db, err = bolt.Open(c.setup.DbPath, FILE_MODE, &bolt.Options{
		Timeout:    10 * time.Second,
		NoGrowSync: false,
	})
	if err != nil {
		glog.Errorln(err)
		return err
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(ipcamsBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(sysBucketName)
		return err
	})
}

func (c *Conf) Close() {
	c.db.Close()
}

func (c *Conf) Get(k []byte) []byte {
	var v []byte
	c.db.View(func(tx *bolt.Tx) error {
		r := tx.Bucket(sysBucketName).Get(k)
		n := len(r)
		if n > 0 {
			v = make([]byte, n)
			copy(v, r)
		}
		return nil
	})
	return v
}

func (c *Conf) Put(k, v []byte) error {
	err := c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(sysBucketName).Put(k, v)
	})
	return err
}

func (c *Conf) Del(k []byte) error {
	err := c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(sysBucketName).Delete(k)
	})
	return err
}

// used by lower ffmpeg
func (c *Conf) GetStuns() []string            { return c.setup.Stuns }
func (c *Conf) GetRecPrefix(id string) string { return path.Join(c.setup.RecDir, id) }
func (c *Conf) GetPingSecond() time.Duration  { return c.setup.PingSecond * time.Second }
func (c *Conf) GetRegToken() []byte           { return c.Get(K_REG_TOKEN) }
func (c *Conf) GetRoomToken() []byte          { return c.Get(K_ROOM_TOKEN) }

func (c *Conf) GetIpcams() (is Ipcams) {
	is = make(Ipcams, 0)
	c.db.View(func(tx *bolt.Tx) error {
		p := tx.Bucket(ipcamsBucketName)
		p.ForEach(func(ik, iv []byte) error {
			if b := p.Bucket(ik); b != nil {
				var i Ipcam
				i.FromBucket(ik, b)
				is[string(ik)] = i
			}
			return nil
		})
		return nil
	})
	return is
}

func (c *Conf) GetIpcam(id []byte) (i Ipcam, err error) {
	err = c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(ipcamsBucketName).Bucket(id)
		if b == nil {
			return ErrIpcamNotFound
		}
		i.FromBucket(id, b)
		return nil
	})
	return i, err
}

// target will trigger remove then create new one
func (c *Conf) PutIpcam(i *Ipcam, target ...[]byte) error {
	err := c.db.Update(func(tx *bolt.Tx) error {
		p := tx.Bucket(ipcamsBucketName)
		if len(target) > 0 && len(target[0]) > 0 {
			if err := p.DeleteBucket(target[0]); err != nil {
				return err
			}
			if i.Id == "" {
				i.Id = string(target[0])
			}
		}
		b, err := p.CreateBucketIfNotExists([]byte(i.Id))
		if err != nil {
			return err
		}
		if err = b.Put(K_IC_URL, []byte(i.Url)); err != nil {
			return err
		}
		if err = b.Put(K_IC_REC, []byte(strconv.FormatBool(i.Rec))); err != nil {
			return err
		}
		if err = b.Put(K_IC_AUDIO_OFF, []byte(strconv.FormatBool(i.AudioOff))); err != nil {
			return err
		}
		if err = b.Put(K_IC_OFF, []byte(strconv.FormatBool(i.Off))); err != nil {
			return err
		}
		if err = b.Put(K_IC_ONLINE, []byte(strconv.FormatBool(i.Online))); err != nil {
			return err
		}
		if err = b.Put(K_IC_HAS_VIDEO, []byte(strconv.FormatBool(i.HasVideo))); err != nil {
			return err
		}
		if err = b.Put(K_IC_HAS_AUDIO, []byte(strconv.FormatBool(i.HasAudio))); err != nil {
			return err
		}
		if err = b.Put(K_IC_WIDTH, []byte(strconv.Itoa(i.Width))); err != nil {
			return err
		}
		if err = b.Put(K_IC_HEIGHT, []byte(strconv.Itoa(i.Height))); err != nil {
			return err
		}
		err = b.Put(K_IC_UPDATE_AT, []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		return err
	})
	return err
}

func (c *Conf) RemoveIpcam(id []byte) error {
	err := c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(ipcamsBucketName).DeleteBucket(id)
	})
	return err
}

func (c *Conf) wsUrl(context string) string { return fmt.Sprintf("%s/one/%s", c.setup.WsUrl, context) }

func (c *Conf) CtrlUrl() string                     { return c.wsUrl("ctrl") }
func (c *Conf) SignalingUrl(reciever string) string { return c.wsUrl("signaling/" + reciever) }
