package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/chrisurwin/alerting-client/agent"
	"github.com/urfave/cli"
)

var VERSION = "v0.1.0-dev"

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "alerting-client"
	app.Version = VERSION
	app.Usage = "Container to monitor Rancher Infrastructure Services"
	app.Action = start
	app.Before = beforeApp
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug,d",
			Usage:  "Debug logging",
			EnvVar: "DEBUG",
		},
		cli.DurationFlag{
			Name:   "poll-interval,i",
			Value:  30 * time.Second,
			Usage:  "Polling interval for checks",
			EnvVar: "POLL_INTERVAL",
		},
		cli.StringFlag{
			Name:   "alert-address,a",
			Usage:  "Alerting server address",
			Value:  "localhost:5050",
			EnvVar: "SERVER_ADDRESS",
		},
		cli.BoolFlag{
			Name:   "k8s,k",
			Usage:  "Specify if environment is a kubernetes environment",
			EnvVar: "K8S",
		},
	}
	app.Run(os.Args)
}

func start(c *cli.Context) {
	if c.String("alert-address") == "" {
		log.Fatal("Alerting server address not set")
	}
	a := agent.NewAgent(c.String("alert-address"), c.Duration("poll-interval"), c.Bool("k8s"))
	a.Start()
}
