package util

import (
	"bytes"
	"sync"
)

type doubleBuffer struct {
	buffs      [2]*bytes.Buffer
	notify     chan struct{}
	rindex     int
	mutex      sync.Mutex
	windex     int
	needNotify bool
}

func newDoubleBuffer() *doubleBuffer {
	pnew := new(doubleBuffer)
	pnew.buffs[0] = new(bytes.Buffer)
	pnew.buffs[1] = new(bytes.Buffer)
	pnew.notify = make(chan struct{}, 1)
	pnew.rindex = 0
	pnew.windex = 1
	pnew.needNotify = false
	return pnew
}

func (db *doubleBuffer) Write(data []byte) {
	db.mutex.Lock()
	db.buffs[db.windex].Write(data)
	needNotify := db.needNotify
	if needNotify {
		db.needNotify = false
	}
	db.mutex.Unlock()
	if needNotify {
		db.notify <- struct{}{}
	}
}

func (db *doubleBuffer) Read() []byte {
	db.buffs[db.rindex].Reset()
	for {
		db.mutex.Lock()
		wlen := db.buffs[db.windex].Len()
		if wlen != 0 {
			tmp := db.windex
			db.windex = db.rindex
			db.rindex = tmp
			db.mutex.Unlock()
			break
		}
		db.needNotify = true
		db.mutex.Unlock()
		<-db.notify
	}
	return db.buffs[db.rindex].Bytes()
}
