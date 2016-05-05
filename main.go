package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sort"
	"time"
)

type config struct {
	Concurrency int
	Server      []string
}

var (
	tlsOkIPList []IP
	conf        config
	cacertPool  *x509.CertPool

	tlsConfig = &tls.Config{
		RootCAs:            cacertPool,
		InsecureSkipVerify: true,
	}
)

func init() {
	configJson, err := ioutil.ReadFile("./config.json")
	checkErr(err)
	err = json.Unmarshal(configJson, &conf)
	cacertFile, err := ioutil.ReadFile("./cacert.pem")
	checkErr(err)
	cacertPool = x509.NewCertPool()
	cacertPool.AppendCertsFromPEM(cacertFile)
}

func checkErr(err error) {
	if err != nil {
		glog.Fatalln(err)
	}
}

func tlsCheckip(ip string, sem chan bool) {
	defer func() { <-sem }()
	dialer := net.Dialer{
		KeepAlive: 0,
		Timeout:   time.Millisecond * 7000,
		DualStack: false,
	}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:443", ip))

	if err != nil {
		glog.Infoln(err)
		return
	}
	defer conn.Close()

	start := time.Now()
	tlsClient := tls.Client(conn, tlsConfig)
	if err = tlsClient.Handshake(); err != nil {
		glog.Infoln(err)
	}
	end := time.Now()

	if tlsClient.ConnectionState().PeerCertificates == nil {
		return
	}

	peerCertSubject := tlsClient.ConnectionState().PeerCertificates[0].Subject
	commonName := peerCertSubject.CommonName
	orgName := peerCertSubject.Organization[0]

	checkedip := IP{
		addr:       ip,
		commonName: commonName,
		orgName:    orgName,
		timeDelay:  int(end.Sub(start).Seconds() * 1000),
	}

	if checkedip.orgName == "Google Inc" {
		if checkedip.commonName == "google.com" {
			checkedip.serverName = "gws"
		} else if checkedip.commonName == "*.c.docs.google.com" || checkedip.commonName == "*.googlevideo.com" {
			checkedip.serverName = "gvs 1.0"
		}
	} else {
		return
	}
	glog.Infof("%s  %s  %s  %d", checkedip.addr, checkedip.commonName, checkedip.serverName, checkedip.timeDelay)

	if strInSlice(checkedip.serverName, conf.Server) {
		tlsOkIPList = append(tlsOkIPList, checkedip)
	}

}

func strInSlice(str string, strslice []string) bool {
	for _, s := range strslice {
		if str == s {
			return true
		}
	}
	return false
}

func main() {
	defer writeOKIPFile()

	fmt.Printf("Concurrency: %d%s", conf.Concurrency, sep)
	fmt.Printf("%#v%s", conf.Server, sep)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			writeOKIPFile()
			os.Exit(1)
		}
	}()
	flag.Set("logtostderr", "true")
	flag.Parse()

	sem := make(chan bool, conf.Concurrency)
	var ipstrlist []string
	ipstrlist = getAllIP("./iprange.txt")
	fmt.Printf("IP to be checked: %d\n", len(ipstrlist))
	time.Sleep(time.Second * 2)

	for _, ip := range ipstrlist {
		sem <- true
		go tlsCheckip(ip, sem)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	println(len(tlsOkIPList))

}

func writeOKIPFile() {
	if len(tlsOkIPList) > 0 {
		okips := &IPList{
			IPList: tlsOkIPList,
		}
		sort.Sort(okips)
		writeOKIP(*okips)
	}
}
