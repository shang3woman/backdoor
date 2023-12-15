package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
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
