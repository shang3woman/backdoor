package main

import (
	"backdoor/util"
	"sync"
)

var gwait WaitSession

type WaitSession struct {
	mutex   sync.Mutex
	uuid    string
	sslconn *util.SSLConn
	notify  chan int
}

func (ws *WaitSession) Wait(uuid string) *util.SSLConn {
	ws.mutex.Lock()
	ws.uuid = uuid
	if ws.notify == nil {
		ws.notify = make(chan int, 1)
	}
	ws.mutex.Unlock()
	<-ws.notify
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	tmpConn := ws.sslconn
	ws.sslconn = nil
	return tmpConn
}

func (ws *WaitSession) IsNeed(uuid string, sslconn *util.SSLConn) bool {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if len(ws.uuid) == 0 {
		return false
	}
	if ws.uuid != uuid {
		return false
	}
	ws.uuid = ""
	ws.sslconn = sslconn
	ws.notify <- 1
	return true
}
