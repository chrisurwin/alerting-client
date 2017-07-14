package agent

import (
	"fmt"
	"net/http"
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

	alert      *alert.AlertSender
	dnsClient  *dns.Client
	httpClient http.Client
	log        *log.Entry
}

func NewAgent(alertAddress string, probePeriod time.Duration, k8s bool) *Agent {

	return &Agent{
		probePeriod: probePeriod,
		k8s:         k8s,
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
			go a.checkHTTP("Etcd Health", "http://etcd.kubernetes.rancher.internal:2379/health")
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
