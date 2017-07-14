package alert

import (
	"os"
)

type Alert struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	SourceHostname string `json:"host"`
}

func New(name, description string) *Alert {
	hostname, _ := os.Hostname()

	return &Alert{
		Name:           name,
		Description:    description,
		SourceHostname: hostname,
	}
}
