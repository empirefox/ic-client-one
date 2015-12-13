package storage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/empirefox/gohome"
	. "github.com/empirefox/ic-client-one/ipcam"
	"github.com/golang/glog"
)

const (
	ID_LEN                = 16
	FILE_MODE os.FileMode = 0644
)

var (
	ErrSystemBucketNotFound = errors.New("system bucket not found")
	ErrSystemKeyNotFound    = errors.New("system key not found")
	ErrIpcamNotFound        = errors.New("ipcam not found")
	ErrHomeDirNotFound      = errors.New("home dir not found")

	// defaults
	appname    = "ic-room"
	dbname     = "room.db"
	recDirName = "ipcam-records"
	pingPeriod = time.Second * 100
	server     = "gocamcom.daoapp.io"
	secure     = true

	sysBucketName    = []byte("system")
	ipcamsBucketName = []byte("ipcams")

	K_REC_DIR     = []byte("RecDir")
	K_REG_TOKEN   = []byte("RegToken")
	K_ROOM_TOKEN  = []byte("RoomToken")
	K_SECURE      = []byte("Secure")
	K_SERVER      = []byte("Server")
	K_PING_PERIOD = []byte("PingPeriod")
)

///////////////////////////////////////////
// Conf
///////////////////////////////////////////
type Conf struct {
	DbPath string
	db     *bolt.DB
}

func NewConf(cpath ...string) Conf {
	p := ""
	if len(cpath) != 0 {
		p = cpath[0]
	}
	return Conf{DbPath: p}
}

// used by lower ffmpeg
func (c *Conf) GetRecPrefix(id string) string {
	return path.Join(c.GetRecDirPath(), id)
}

func (c *Conf) dbPath() string {
	if c.DbPath != "" {
		return c.DbPath
	}
	return path.Join(gohome.Config(appname), dbname)
}

func (c *Conf) Open() (err error) {
	if gohome.Home() == "" {
		return ErrHomeDirNotFound
	}

	cpath := c.dbPath()
	err = os.MkdirAll(path.Dir(cpath), os.ModePerm)
	if err != nil {
		glog.Errorln(err)
		return err
	}

	c.db, err = bolt.Open(cpath, FILE_MODE, &bolt.Options{
		Timeout:    10 * time.Second,
		NoGrowSync: false,
	})
	if err != nil {
		glog.Errorln(err)
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(ipcamsBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(sysBucketName)
		return err
	})
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.GetRecDirPath(), os.ModePerm)
	return err
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

func (c *Conf) GetPingPeriod() time.Duration {
	r, err := time.ParseDuration(string(c.Get(K_PING_PERIOD)))
	if err != nil {
		return pingPeriod
	}
	return r
}

func (c *Conf) GetRegToken() []byte {
	return c.Get(K_REG_TOKEN)
}

func (c *Conf) GetRoomToken() []byte {
	return c.Get(K_ROOM_TOKEN)
}

func (c *Conf) GetServer() string {
	r := string(c.Get(K_SERVER))
	if r != "" {
		return r
	}
	return server
}

// default true
func (c *Conf) IsSecure() bool {
	s, err := strconv.ParseBool(string(c.Get(K_SECURE)))
	if err != nil {
		return secure
	}
	return s
}

// support env like ${var} or $var
func (c *Conf) GetRecDirPath() string {
	dir := string(c.Get(K_REC_DIR))
	if dir == "" {
		dir = recDirName
	}
	dir = os.ExpandEnv(dir)
	if path.IsAbs(dir) {
		return dir
	}
	return path.Join(gohome.Home(), dir)
}

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

func (c *Conf) wsUrl(context string) string {
	p := "ws"
	if c.IsSecure() {
		p = "wss"
	}
	return fmt.Sprintf("%s://%s/one/%s", p, c.GetServer(), context)
}

func (c *Conf) CtrlUrl() string {
	return c.wsUrl("ctrl")
}

func (c *Conf) SignalingUrl(reciever string) string {
	return c.wsUrl("signaling/" + reciever)
}

func (c *Conf) RegRoomUrl() string {
	p := "http"
	if c.IsSecure() {
		p = "https"
	}
	return fmt.Sprintf("%s://%s/%s", p, c.GetServer(), "one-rest/reg-room")
}
