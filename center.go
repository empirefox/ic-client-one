package main

import "encoding/json"

type Center struct {
	status               string
	statusReciever       map[*Connection]bool
	AddStatusReciever    chan *Connection
	RemoveStatusReciever chan *Connection
	ChangeStatus         chan string
	Quit                 chan bool
	CtrlConn             *Connection
}

func NewCenter() *Center {
	return &Center{
		statusReciever:       make(map[*Connection]bool),
		AddStatusReciever:    make(chan *Connection),
		RemoveStatusReciever: make(chan *Connection),
		ChangeStatus:         make(chan string),
		Quit:                 make(chan bool),
	}
}

func (center *Center) Run() {
	for {
		select {
		case c := <-center.AddStatusReciever:
			center.statusReciever[c] = true
		case c := <-center.RemoveStatusReciever:
			if _, ok := center.statusReciever[c]; ok {
				delete(center.statusReciever, c)
				close(c.Send)
			}
		case center.status = <-center.ChangeStatus:
			status, err := center.GetStatus()
			if err != nil {
				continue
			}
			for c := range center.statusReciever {
				select {
				case c.Send <- status:
				default:
					close(c.Send)
					delete(center.statusReciever, c)
				}
			}
		case <-center.Quit:
			return
		}
	}
}

func (center *Center) Close() {
	close(center.Quit)
}

func (center *Center) GetStatus() ([]byte, error) {
	statusMap := map[string]string{"type": "Status", "content": center.status}
	return json.Marshal(statusMap)
}

func (center *Center) AddCtrlConn(c *Connection) {
	center.CtrlConn = c
	center.AddStatusReciever <- c
}

func (center *Center) RemoveCtrlConn() {
	center.RemoveStatusReciever <- center.CtrlConn
	center.CtrlConn = nil
}
