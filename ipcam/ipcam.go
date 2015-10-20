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
	K_IC_URL       = []byte("Url")
	K_IC_REC       = []byte("Rec")
	K_IC_AUDIO_OFF = []byte("AudioOff")
	K_IC_OFF       = []byte("Off")
	K_IC_ONLINE    = []byte("Online")
	K_IC_HAS_VIDEO = []byte("HasVideo")
	K_IC_HAS_AUDIO = []byte("HasAudio")
	K_IC_WIDTH     = []byte("Width")
	K_IC_HEIGHT    = []byte("Height")
	K_IC_UPDATE_AT = []byte("UpdatedAt")
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
	Id        string `json:",omitempty" structs:",omitempty" view:",omitempty"`
	Url       string `json:",omitempty" structs:",omitempty" view:"-"`
	Rec       bool   `json:",omitempty" structs:",omitempty" view:"-"`
	AudioOff  bool   `json:",omitempty" structs:",omitempty" view:"-"`
	Off       bool   `json:",omitempty" structs:",omitempty" view:",omitempty"`
	Online    bool   `json:",omitempty" structs:",omitempty" view:",omitempty"`
	HasVideo  bool   `json:",omitempty" structs:",omitempty" view:",omitempty"`
	HasAudio  bool   `json:",omitempty" structs:",omitempty" view:",omitempty"`
	Width     int    `json:",omitempty" structs:",omitempty" view:",omitempty"`
	Height    int    `json:",omitempty" structs:",omitempty" view:",omitempty"`
	UpdatedAt int64  `json:",omitempty" structs:",omitempty" view:",omitempty"`
}

func (i *Ipcam) FromBucket(id []byte, b *bolt.Bucket) {
	i.Id = string(id)
	i.Url = string(b.Get(K_IC_URL))
	i.Rec, _ = strconv.ParseBool(string(b.Get(K_IC_REC)))
	i.AudioOff, _ = strconv.ParseBool(string(b.Get(K_IC_AUDIO_OFF)))
	i.Off, _ = strconv.ParseBool(string(b.Get(K_IC_OFF)))
	i.Online, _ = strconv.ParseBool(string(b.Get(K_IC_ONLINE)))
	i.HasVideo, _ = strconv.ParseBool(string(b.Get(K_IC_HAS_VIDEO)))
	i.HasAudio, _ = strconv.ParseBool(string(b.Get(K_IC_HAS_AUDIO)))
	i.Width, _ = strconv.Atoi(string(b.Get(K_IC_WIDTH)))
	i.Height, _ = strconv.Atoi(string(b.Get(K_IC_HEIGHT)))
	i.UpdatedAt, _ = strconv.ParseInt(string(b.Get(K_IC_UPDATE_AT)), 10, 64)
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
