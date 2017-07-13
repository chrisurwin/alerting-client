package agent

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/chrisurwin/alerting-client/healthcheck"

	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
)

var (
	serverHostname = ""
	serverPort     = ""
	logLevel       = ""
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

//StartAgent function to start the alerting agent
func StartAgent(hostname string, port string, poll string, k8s string) {

	serverHostname = hostname
	serverPort = port

	go healthcheck.StartHealthcheck()
	//go startHealthcheck()

	for {
		dnsCheck("MetaData DNS", "rancher-metadata.rancher.internal")
		httpCheck("Rancher Metadata", "http://169.254.169.250")

		if k8s != "false" {
			httpCheck("Kube API", "http://kubernetes.kubernetes.rancher.internal")
			httpCheck("Etcd Health", "http://etcd.kubernetes.rancher.internal:2379/health")
		}

		interval, err := strconv.Atoi(poll)
		if err != nil {
			interval = 30
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func dnsCheck(checkName string, target string) {
	server := "169.254.169.250"

	c := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion(target+".", dns.TypeA)
	r, t, err := c.Exchange(&m, server+":53")
	if err != nil {
		logrus.Error(err)
		alert(checkName, err.Error())
		return
	}
	if len(r.Answer) == 0 {
		logrus.Error(checkName + ":No results")
	} else {
		logrus.Info(checkName + ":" + target + " succeeded in " + t.String())
	}
}

func httpCheck(checkName string, address string) {
	timeout := time.Duration(2 * time.Second)

	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(address)

	if resp != nil {

		if (resp.StatusCode != 200) || (err != nil) {
			logrus.Error(checkName + ":Check failed")
			alert(checkName, "Received the following non-200 response:"+strconv.Itoa(resp.StatusCode))
		} else {
			logrus.Info(checkName + " succeeded")
		}
		resp.Body.Close()
	} else {

		logrus.Error(checkName + ":Timeout")
		alert(checkName, "Timeout on operation")
	}

}

func alert(name string, description string) {
	host, _ := os.Hostname()
	url := "http://" + serverHostname + ":" + serverPort + "/report_alert"

	if logLevel == "DEBUG" {
		logrus.Info("URL :> " + url)
	}

	var jsonStr = []byte(`{"name": "` + name + `", "description":"` + description + `", "host": "` + host + `"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Content-Type", "application/json")

	timeout := time.Duration(4 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if resp != nil {
		if (resp.StatusCode != 200) || (err != nil) {
			logrus.Error("Failed to send to server")
			if err != nil {
				panic(err)
			}
		}
		resp.Body.Close()
	} else {
		logrus.Info("Timeout sending to server")
	}

	if logLevel == "DEBUG" {
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}

}
