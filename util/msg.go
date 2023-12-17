package util

const (
	CMD_CHANNEL = iota
	SOCKS5_CHANNEL
)

const (
	CMD_FILE_BEG = iota
	CMD_FILE_DATA
	CMD_FILE_END
	CMD_DOWNLOAD_REQ
	CMD_CD
	CMD_NEWPROCESS_EXEC
	CMD_EXEC
	CMD_SET_SHELL
	CMD_GET_ENV
	CMD_SET_ENV
	CMD_PRINT
	CMD_PWD
)

const (
	SOCKS5_CONNECT = iota
	SOCKS5_CLOSE
	SOCKS5_DATA
)

type Info struct {
	HostName string `json:"hostname,omitempty"`
	LocalIP  string `json:"localip,omitempty"`
	PID      int    `json:"pid,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	OSType   string `json:"ostype,omitempty"`
	Time     string `json:"-"`
}

type Request struct {
	Magic string `json:"magic,omitempty"`
	Cmd   string `json:"cmd,omitempty"`
}
