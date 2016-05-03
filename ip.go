package main

import (
	"bytes"
	"net"
	"sort"
	"strings"
)

type IP struct {
	addr       string
	commonName string
	orgName    string
	serverName string
	timeDelay  int
}

type IPList struct {
	sort.Interface
	IPList []IP
}

func (iplist IPList) Len() int {
	return len(iplist.IPList)
}

func (iplist IPList) Less(i, j int) bool {
	if iplist.IPList[i].timeDelay <= iplist.IPList[j].timeDelay {
		return true
	}

	return false
}

func (iplist *IPList) Swap(i, j int) {
	t := *new(IP)
	t = (*iplist).IPList[i]
	(*iplist).IPList[i] = (*iplist).IPList[j]
	(*iplist).IPList[j] = t
}

func inc(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func parseIPRange(iprange string) []string {
	var ipStrList []string
	if strings.Contains(iprange, "-") {
		parts := strings.Split(iprange, "-")
		startip := net.ParseIP(parts[0])
		endip := net.ParseIP(parts[1])
		for ip := startip; bytes.Compare(ip, endip) <= 0; inc(ip) {
			ipStrList = append(ipStrList, ip.String())
		}
		if strings.HasSuffix(ipStrList[0], ".0") {
			ipStrList = ipStrList[1:]
		}
		if strings.HasSuffix(ipStrList[len(ipStrList)-1], ".0") {
			ipStrList = ipStrList[:len(ipStrList)-2]
		}
		return ipStrList

	} else if strings.Contains(iprange, "/") {
		ip, ipnet, err := net.ParseCIDR(iprange)
		checkErr(err)
		for ip = ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			ipStrList = append(ipStrList, ip.String())
		}
		return ipStrList[1 : len(ipStrList)-1]
	} else {
		ipStrList = append(ipStrList, iprange)
		return ipStrList
	}

}

func getAllIP(file string) []string {
	alliprange := readIPFile(file)
	var allip []string
	for _, iprange := range alliprange {
		allip = append(allip, parseIPRange(iprange)...)
	}
	return allip
}
