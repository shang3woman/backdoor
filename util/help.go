package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func SendCmdMsg(conn *SSLConn, cmd byte, data []byte) {
	var buffer bytes.Buffer
	buffer.WriteByte(CMD_CHANNEL)
	buffer.WriteByte(cmd)
	if len(data) != 0 {
		buffer.Write(data)
	}
	conn.Write(buffer.Bytes())
}

func SendSocksMsg(conn *SSLConn, cmd byte, sid uint32, data []byte) {
	var buffer bytes.Buffer
	buffer.WriteByte(SOCKS5_CHANNEL)
	buffer.WriteByte(cmd)
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], sid)
	buffer.Write(tmp[:])
	if len(data) != 0 {
		buffer.Write(data)
	}
	conn.Write(buffer.Bytes())
}

func storeFile(fileName string, fileData []byte) error {
	pfile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer pfile.Close()
	_, err = pfile.Write(fileData)
	return err
}

func TryStoreFileForMulti(fileName string, fileData []byte) error {
	var err error
	err = storeFile(fileName, fileData)
	if err == nil {
		return nil
	}
	for i := 1; i < 10; i++ {
		tmpName := fmt.Sprintf("%s.%d", fileName, i)
		err = storeFile(tmpName, fileData)
		if err == nil {
			break
		}
	}
	return err
}

func ParseIP(str string) (string, uint16, bool) {
	str = strings.TrimSpace(str)
	strArr := strings.Split(str, ":")
	if len(strArr) != 2 {
		return "", 0, false
	}
	ip := net.ParseIP(strArr[0])
	if ip == nil || ip.To4() == nil {
		return "", 0, false
	}
	port, err := strconv.ParseUint(strArr[1], 10, 16)
	if err != nil {
		return "", 0, false
	}
	if port == 0 {
		return "", 0, false
	}
	return ip.String(), uint16(port), true
}
