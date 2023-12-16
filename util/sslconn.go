package util

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"
)

type SSLConn struct {
	conn     *ConnWrap
	mutex    sync.Mutex
	pencoder *Encoder
	pdecoder *Decoder
}

func NewSSLConn(conn net.Conn) *SSLConn {
	pnew := new(SSLConn)
	pnew.conn = NewConnWrap(conn)
	pnew.pencoder = NewEncoder()
	pnew.pdecoder = NewDecoder()
	return pnew
}

func (sslconn *SSLConn) Read() ([]byte, error) {
	sslconn.mutex.Lock()
	defer sslconn.mutex.Unlock()
	var header [4]byte
	if _, err := io.ReadFull(sslconn.conn, header[:]); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(header[:])
	if length == 0 {
		return []byte{}, nil
	}
	msg := make([]byte, length)
	if _, err := io.ReadFull(sslconn.conn, msg); err != nil {
		return nil, err
	}
	return sslconn.pdecoder.Decode(msg)
}

func (sslconn *SSLConn) Write(data []byte) (int, error) {
	sslconn.mutex.Lock()
	defer sslconn.mutex.Unlock()
	if len(data) != 0 {
		data = sslconn.pencoder.Encode(data)
	}
	msg := binary.LittleEndian.AppendUint32(nil, uint32(len(data)))
	if len(data) != 0 {
		msg = append(msg, data...)
	}
	return sslconn.conn.Write(msg)
}

func (sslconn *SSLConn) Close() error {
	return sslconn.conn.Close()
}

func (sslconn *SSLConn) LocalAddr() net.Addr {
	return sslconn.conn.LocalAddr()
}

func (sslconn *SSLConn) RemoteAddr() net.Addr {
	return sslconn.conn.RemoteAddr()
}

func (sslconn *SSLConn) SetDeadline(t time.Time) error {
	return sslconn.conn.SetDeadline(t)
}

func (sslconn *SSLConn) SetReadDeadline(t time.Time) error {
	return sslconn.conn.SetReadDeadline(t)
}

func (sslconn *SSLConn) SetWriteDeadline(t time.Time) error {
	return sslconn.conn.SetWriteDeadline(t)
}
