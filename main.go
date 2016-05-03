package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"
)

type config struct {
	Concurrency int
	Httptimeout int
	Server      []string
}

var (
	httpOkIPList []IP
	conf         config
	cacertPool   *x509.CertPool
	tlsConfig    = &tls.Config{
		RootCAs:            cacertPool,
		InsecureSkipVerify: true,
	}

	httpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
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
	httpClient.Timeout = time.Millisecond * time.Duration(conf.Httptimeout)
}

func checkErr(err error) {
	if err != nil {
		glog.Fatalln(err)
	}
}

func httpCheckip(ip string, sem chan bool) {
	defer func() { <-sem }()
	start := time.Now()
	resp, err := httpClient.Head(fmt.Sprintf("https://%s", ip))
	end := time.Now()

	if err != nil {
		if !strings.Contains(err.Error(), "www.google.com") {
			glog.Infof("%s  %s", ip, err)
			return
		}
		checkedip := IP{
			addr:       ip,
			commonName: "google.com",
			orgName:    "Google Inc",
			serverName: "gws",
			timeDelay:  int(end.Sub(start).Seconds() * 1000),
		}
		glog.Infof("%s  %s  %d", checkedip.addr, checkedip.serverName, checkedip.timeDelay)

		if strInSlice(checkedip.serverName, conf.Server) {
			httpOkIPList = append(httpOkIPList, checkedip)
		}
		return
	}

	peerCertSubject := resp.TLS.PeerCertificates[0].Subject
	commonName := peerCertSubject.CommonName
	orgName := ""
	if len(peerCertSubject.Organization) > 0 {
		orgName = peerCertSubject.Organization[0]
	}
	serverName := resp.Header.Get("Server")
	checkedip := IP{
		addr:       ip,
		commonName: commonName,
		orgName:    orgName,
		serverName: serverName,
		timeDelay:  int(end.Sub(start).Seconds() * 1000),
	}

	glog.Infof("%s  %s  %d", checkedip.addr, checkedip.serverName, checkedip.timeDelay)

	if strInSlice(checkedip.serverName, conf.Server) {
		httpOkIPList = append(httpOkIPList, checkedip)
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

	fmt.Printf("Concurrency: %d%sHttptimeout: %d%s", conf.Concurrency, sep, conf.Httptimeout, sep)
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
		go httpCheckip(ip, sem)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	println(len(httpOkIPList))

}

func writeOKIPFile() {
	if len(httpOkIPList) > 0 {
		okips := &IPList{
			IPList: httpOkIPList,
		}
		sort.Sort(okips)
		writeOKIP(*okips)
	}
}
