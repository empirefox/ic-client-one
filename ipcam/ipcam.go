// seperate to use ffjson fast
package ipcam

import (
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/fatih/structs"
)

const (
	TAG_ALL  = "structs"
	TAG_VIEW = "view"
)

var (
	K_IC_URL    = []byte("Url")
	K_IC_REC    = []byte("Rec")
	K_IC_OFF    = []byte("Off")
	K_IC_ONLINE = []byte("Online")
)

type Ipcams map[string]Ipcam

func (is Ipcams) Map(tag ...string) map[string]map[string]interface{} {
	r := make(map[string]map[string]interface{}, len(is))
	for k, v := range is {
		r[k] = v.Map(tag...)
	}
	return r
}

// except json, other tags are used by structs when encoding(not decoding)
type Ipcam struct {
	Id     string `json:"id,omitempty"     structs:"id,omitempty"     view:"id,omitempty"`
	Url    string `json:"url"              structs:"url,omitempty"    view:"-"`
	Rec    bool   `json:"rec,omitempty"    structs:"rec,omitempty"    view:"-"`
	Off    bool   `json:"off,omitempty"    structs:"off,omitempty"    view:"off,omitempty"`
	Online bool   `json:"online,omitempty" structs:"online,omitempty" view:"online,omitempty"`
}

func (i *Ipcam) FromBucket(id []byte, b *bolt.Bucket) {
	i.Id = string(id)
	i.Url = string(b.Get(K_IC_URL))
	i.Rec, _ = strconv.ParseBool(string(b.Get(K_IC_REC)))
	i.Off, _ = strconv.ParseBool(string(b.Get(K_IC_OFF)))
	i.Online, _ = strconv.ParseBool(string(b.Get(K_IC_ONLINE)))
}

func (i *Ipcam) Map(tag ...string) map[string]interface{} {
	ss := structs.New(i)
	if len(tag) != 0 {
		ss.TagName = tag[0]
	} else {
		ss.TagName = TAG_ALL
	}
	return ss.Map()
}

// only unmarshal
type SetterIpcam struct {
	Target string `json:"target,omitempty"`
	Ipcam
}
