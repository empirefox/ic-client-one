package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/empirefox/ic-client-one/ipcam"
)

const jsonContent = `{
  "DbPath": "%s",
  "RecDir": "/tmp/ic-client-one-rec-dir",
  "WsUrl": "ws://127.0.0.1:9998",
  "PingSecond": 50,
  "Stuns": [
    "stun3.l.google.com:19302",
    "stun.ideasip.com",
    "stun4.l.google.com:19302",
    "stun2.l.google.com:19302",
    "stun1.l.google.com:19302",
    "stun.ekiga.net",
    "stun.schlund.de",
    "stun.voipstunt.com",
    "stun.voiparound.com",
    "stun.voipbuster.com",
    "stun.voxgratia.org"
  ]
	}`

type TestConf struct {
	*Conf
}

func newSetup() string {
	return fmt.Sprintf(jsonContent, tempfile())
}

func NewTestConf() *TestConf {
	c, err := NewConf(newSetup())
	if err != nil {
		panic(err)
	}
	if err := c.Open(); err != nil {
		panic(err)
	}
	return &TestConf{c}
}

func (c *TestConf) Close() {
	defer os.Remove(c.db.Path())
	c.Conf.Close()
}

// tempfile returns a temporary file path.
func tempfile() string {
	f, _ := ioutil.TempFile("", "ic-client-one-db-")
	f.Close()
	os.Remove(f.Name())
	return f.Name()
}

func TestConf_Open(t *testing.T) {
	c, err := NewConf(newSetup())
	if err != nil {
		panic(err)
	}
	if err := c.Open(); err != nil {
		t.Errorf("NewConf failed, err: %v\n", err)
	}
	defer os.Remove(c.db.Path())
	defer c.Close()

	c.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(sysBucketName); b == nil {
			t.Errorf("No sys bucket created after opened")
		}
		if b := tx.Bucket(ipcamsBucketName); b == nil {
			t.Errorf("No ipcams bucket created after opened")
		}
		return nil
	})
}

func TestConf_PutGet(t *testing.T) {
	c := NewTestConf()
	defer c.Close()

	if err := c.Put([]byte("a"), []byte("b")); err != nil {
		t.Errorf("failed to put value to db, err: %v\n", err)
	}

	if r := string(c.Get([]byte("a"))); r != "b" {
		t.Errorf("failed to get value from db\n")
	}
}

func TestConf_PutGetIpcam(t *testing.T) {
	c := NewTestConf()
	defer c.Close()

	i := &ipcam.Ipcam{Url: "aurl"}
	if err := c.PutIpcam(i); err == nil {
		t.Errorf("shoud get error when no id specialed: %v\n", err)
	}

	i.Id = "aid"
	if err := c.PutIpcam(i); err != nil {
		t.Errorf("shoud not get error when putting ipcam with inline id: %v\n", err)
	}

	i2, err := c.GetIpcam([]byte("aid"))
	if err != nil {
		t.Errorf("should get ipcam from db, err: %v\n", err)
	}
	if i2.Url != "aurl" {
		t.Errorf("should get correct ipcam.url from db\n")
	}
}

func TestConf_ChangeRemoveIpcam(t *testing.T) {
	c := NewTestConf()
	defer c.Close()

	i := &ipcam.Ipcam{Id: "aid", Url: "aurl"}
	if err := c.PutIpcam(i); err != nil {
		t.Errorf("shoud not get error when putting ipcam with inline id: %v\n", err)
	}

	i.Id = "bid"
	if err := c.PutIpcam(i, []byte("aid")); err != nil {
		t.Errorf("shoud not get error when changing ipcam with inline id: %v\n", err)
	}

	i2, err := c.GetIpcam([]byte("bid"))
	if err != nil {
		t.Errorf("should get ipcam from db, err: %v\n", err)
	}
	if i2.Url != "aurl" {
		t.Errorf("should get correct ipcam.url from db\n")
	}

	if err := c.RemoveIpcam([]byte("aid")); err == nil {
		t.Errorf("should get error when remove non-exist ipcam\n")
	}

	if err := c.RemoveIpcam([]byte("bid")); err != nil {
		t.Errorf("should remove ipcam from db, err: %v\n", err)
	}

	if _, err := c.GetIpcam([]byte("bid")); err == nil {
		t.Errorf("should get error after removing, err: %v\n", err)
	}
}
