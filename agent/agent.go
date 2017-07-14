package agent

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/chrisurwin/alerting-client/alert"
	"github.com/chrisurwin/alerting-client/healthcheck"
	"github.com/miekg/dns"
)

const (
	RancherEndpointDNS = "169.254.169.250:53"
)

type Agent struct {
	sync.WaitGroup

	probePeriod time.Duration
	k8s         bool

	alert      *alert.Sender
	dnsClient  *dns.Client
	httpClient http.Client
	log        *log.Entry
}

func NewAgent(alertAddress string, probePeriod time.Duration) *Agent {

	return &Agent{
		probePeriod: probePeriod,
		k8s:         checkK8S(),
		alert:       alert.NewSender(alertAddress, 4),
		dnsClient:   &dns.Client{},
		httpClient: http.Client{
			Timeout: time.Duration(2 * time.Second),
		},
		log: log.WithField("pkg", "agent"),
	}
}

func (a *Agent) Start() {
	go healthcheck.StartHealthcheck()

	t := time.NewTicker(a.probePeriod)
	for _ = range t.C {
		a.log.Debug("Probing infrastructure.")

		go a.checkDNS("MetaData DNS", "rancher-metadata.rancher.internal")
		go a.checkHTTP("Rancher Metadata", "http://169.254.169.250")

		if a.k8s {
			var etcd = getDNS("http://etcd.kubernetes.rancher.internal:2379/health")
			if len(etcd) > 0 {
				for i := range etcd {
					go a.checkHTTP("Etcd Health", etcd[i])
				}
			}
			go a.checkHTTP("Kube API", "http://kubernetes.kubernetes.rancher.internal")
		}

		a.Wait()
	}
}

func (a *Agent) checkDNS(checkName, target string) {
	a.Add(1)
	defer a.Done()

	m := dns.Msg{}
	m.SetQuestion(target+".", dns.TypeA)
	r, t, err := a.dnsClient.Exchange(&m, RancherEndpointDNS)
	if err != nil {
		a.alert.Send(alert.New(checkName, err.Error()))
		return
	}
	if len(r.Answer) == 0 {
		a.log.Error(checkName + ":No results")
	} else {
		a.log.Info(checkName + ":" + target + " succeeded in " + t.String())
	}
}

func getDNS(target string) []string {

	var empty []string
	u, err := url.Parse(target)
	if err != nil {
		log.Error(err)
		return empty
	}
	host, port, _ := net.SplitHostPort(u.Host)

	c := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion(host+".", dns.TypeA)
	r, _, err := c.Exchange(&m, "8.8.8.8:53")
	if err != nil {
		log.Fatal(err)
	}
	if len(r.Answer) == 0 {
		log.Error(target + ":No results")
		return empty
	}

	ip := make([]string, len(r.Answer))
	var i = 0
	for _, ans := range r.Answer {
		Arecord := ans.(*dns.A)
		ip[i] = u.Scheme + "://" + Arecord.A.String() + ":" + port + u.Path
		i++
	}
	return ip
}

func (a *Agent) checkHTTP(checkName, address string) {
	a.Add(1)
	defer a.Done()

	resp, err := a.httpClient.Get(address)
	if err != nil {
		a.alert.Send(alert.New(checkName, err.Error()))
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		a.alert.Send(alert.New(checkName, fmt.Sprintf("Expecting HTTP 2XX, received '%s'", resp.Status)))
	default:
		a.log.WithField("check", "success").Debugf("Received '%s'", resp.Status)
	}
}

func checkK8S() bool {
	httpClient :=
		http.Client{
			Timeout: time.Duration(2 * time.Second),
		}
	resp, err := httpClient.Get("http://rancher-metadata/latest/stacks/kubernetes")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == 200:
		return true
	default:
		return false
	}
}
