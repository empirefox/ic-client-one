package ipcam

import "testing"

func TestIpcam_Map(t *testing.T) {
	f := Ipcam{}
	f.Id = "myid"
	f.Url = "myurl"
	f.Rec = true
	f.Off = true
	f.Online = false

	m := f.Map()
	_, ok1 := m["id"]
	_, ok2 := m["url"]
	_, ok3 := m["rec"]
	_, ok4 := m["off"]
	_, ok5 := m["online"]
	if !ok1 || !ok2 || !ok3 || !ok4 || ok5 {
		t.Errorf("should get default output\n")
	}

	m = f.Map(TAG_VIEW)
	_, ok1 = m["id"]
	_, ok2 = m["url"]
	_, ok3 = m["rec"]
	_, ok4 = m["off"]
	_, ok5 = m["online"]
	if !ok1 || ok2 || ok3 || !ok4 || ok5 {
		t.Errorf("should get default output\n")
	}
}
