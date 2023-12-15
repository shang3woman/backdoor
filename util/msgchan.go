package util

import "sync"

type MsgChan struct {
	notify     chan struct{}
	mutex      sync.Mutex
	needNotify bool
	msgs       [][]byte
	exit       bool
}

func NewMsgChan() *MsgChan {
	pnew := new(MsgChan)
	pnew.notify = make(chan struct{}, 1)
	pnew.needNotify = false
	pnew.exit = false
	return pnew
}

func (ch *MsgChan) In(msg []byte) {
	if len(msg) == 0 {
		return
	}
	ch.mutex.Lock()
	if ch.exit {
		ch.mutex.Unlock()
		return
	}
	ch.msgs = append(ch.msgs, msg)
	needNotify := ch.needNotify
	if needNotify {
		ch.needNotify = false
	}
	ch.mutex.Unlock()
	if needNotify {
		ch.notify <- struct{}{}
	}
}

func (ch *MsgChan) Out() ([]byte, bool) {
	for {
		ch.mutex.Lock()
		if ch.exit {
			ch.mutex.Unlock()
			return nil, false
		}
		length := len(ch.msgs)
		if length == 0 {
			ch.needNotify = true
			ch.mutex.Unlock()
			<-ch.notify
			continue
		}
		tmp := ch.msgs[0]
		for i := 1; i < length; i++ {
			ch.msgs[i-1] = ch.msgs[i]
		}
		ch.msgs[length-1] = nil
		ch.msgs = ch.msgs[0 : length-1]
		ch.mutex.Unlock()
		return tmp, true
	}
}

func (ch *MsgChan) Close() {
	ch.mutex.Lock()
	if ch.exit {
		ch.mutex.Unlock()
		return
	}
	ch.msgs = nil
	ch.exit = true
	needNotify := ch.needNotify
	if needNotify {
		ch.needNotify = false
	}
	ch.mutex.Unlock()
	if needNotify {
		ch.notify <- struct{}{}
	}
}
