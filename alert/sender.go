package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Sender struct {
	alertEndpoint string
	alerts        chan *Alert
	httpClient    http.Client
	log           *log.Entry
}

func NewSender(alertAddress string, bufferSize int) *Sender {
	s := &Sender{
		alertEndpoint: fmt.Sprintf("http://%s/report_alert", alertAddress),
		alerts:        make(chan *Alert, bufferSize),
		httpClient: http.Client{
			Timeout: 2 * time.Second,
		},
		log: log.WithField("pkg", "alert_sender"),
	}
	go s.watch()
	return s
}

func (s *Sender) Send(a *Alert) {
	s.alerts <- a
}

func (s *Sender) watch() {
	for a := range s.alerts {
		l := s.log.WithField("name", a.Name)
		l.Debug("Received alert")

		if err := s.send(a); err != nil {
			l.WithField("error", err.Error()).Error("Could not send alert")
		}

		l.Debugf("Chan alerts at %d%% capacity (%d/%d)",
			len(s.alerts)*100/cap(s.alerts), len(s.alerts), cap(s.alerts))
	}
}

func (s *Sender) send(a *Alert) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", s.alertEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Server return HTTP %d", resp.StatusCode)
	}

	if log.GetLevel() == log.DebugLevel {
		s.log.Debugf("response Status: %s", resp.Status)
		s.log.Debugf("response Headers: %v", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		s.log.Debugf("response Body: %s", string(body))
	}
	return nil
}
