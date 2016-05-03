package main

import (
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

const (
	defaultFileMode = 0644
)

var sep string

func init() {
	if runtime.GOOS == "windows" {
		sep = "\r\n"
	} else {
		sep = "\n"
	}
}

func readIPFile(file string) []string {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		glog.Fatal(err)
	}

	return strings.Split(string(bytes), sep)
}

func writeOKIP(okIPList IPList) {
	f, err := os.Create("./okip.txt")
	checkErr(err)
	defer f.Close()
	for _, ip := range okIPList.IPList {
		f.WriteString(fmt.Sprintf("%s %d %s %s%s", ip.addr, ip.timeDelay, ip.commonName, ip.serverName, sep))
	}
}
