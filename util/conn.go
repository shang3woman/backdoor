package util

import (
	"net"
	"sync"
	"time"
)

type ConnWrap struct {
	conn    net.Conn
	dbuffer *doubleBuffer

	mutex sync.Mutex
	err   error
}

func NewConnWrap(conn net.Conn) *ConnWrap {
	pwrap := new(ConnWrap)
	pwrap.conn = conn
	pwrap.dbuffer = newDoubleBuffer()
	go pwrap.loopSend()
	return pwrap
}

func (wrap *ConnWrap) LoadError() error {
	var err error
	wrap.mutex.Lock()
	err = wrap.err
	wrap.mutex.Unlock()
	return err
}

func (wrap *ConnWrap) StoreError(err error) {
	wrap.mutex.Lock()
	wrap.err = err
	wrap.mutex.Unlock()
}

func (wrap *ConnWrap) Read(b []byte) (n int, err error) {
	return wrap.conn.Read(b)
}

func (wrap *ConnWrap) Write(b []byte) (n int, err error) {
	tmperr := wrap.LoadError()
	if tmperr != nil {
		return 0, tmperr
	}
	wrap.dbuffer.Write(b)
	return len(b), nil
}

func doWrite(conn net.Conn, data []byte) error {
	writed := 0
	for writed < len(data) {
		n, err := conn.Write(data[writed:])
		if err != nil {
			return err
		}
		writed += n
	}
	return nil
}

func (wrap *ConnWrap) loopSend() {
	for {
		data := wrap.dbuffer.Read()
		err := doWrite(wrap.conn, data)
		if err == nil {
			continue
		}
		wrap.StoreError(err)
		break
	}
}

func (wrap *ConnWrap) Close() error {
	err := wrap.conn.Close()
	wrap.Write([]byte{0})
	return err
}

func (wrap *ConnWrap) LocalAddr() net.Addr {
	return wrap.conn.LocalAddr()
}

func (wrap *ConnWrap) RemoteAddr() net.Addr {
	return wrap.conn.RemoteAddr()
}

func (wrap *ConnWrap) SetDeadline(t time.Time) error {
	return wrap.conn.SetDeadline(t)
}

func (wrap *ConnWrap) SetReadDeadline(t time.Time) error {
	return wrap.conn.SetReadDeadline(t)
}

func (wrap *ConnWrap) SetWriteDeadline(t time.Time) error {
	return wrap.conn.SetWriteDeadline(t)
}
