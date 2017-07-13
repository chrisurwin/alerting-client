package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/chrisurwin/alerting-agent/agent"
	"os"
)

var VERSION = "v0.1.0-dev"

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "alerting-agent"
	app.Version = VERSION
	app.Usage = "Container to monitor Rancher Infrastructure Services"
	app.Action = start
	app.Before = beforeApp
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug,d",
			Usage: "Debug logging",
		},
		cli.StringFlag{
			Name:   "poll-interval",
			Value:  "30",
			Usage:  "Polling interval for check",
			EnvVar: "POLL_INTERVAL",
		},
		cli.StringFlag{
			Name:   "server-hostname",
			Usage:  "Alerting server",
			Value:  "localhost",
			EnvVar: "SERVER_HOSTNAME",
		},
		cli.StringFlag{
			Name:   "server-port",
			Value:  "5050",
			Usage:  "Alerting server port",
			EnvVar: "SERVER_PORT",
		},
		cli.StringFlag{
			Name:   "k8s",
			Value:  "false",
			Usage:  "Specify if environment is a kubernetes environment",
			EnvVar: "K8S",
		},
	}
	app.Run(os.Args)

}

func start(c *cli.Context) {

	if c.GlobalString("server-hostname") == "" {
		logrus.Fatalf("SERVER_HOSTNAME not set")
	}
	for {
		agent.StartAgent(c.GlobalString("server-hostname"), c.GlobalString("server-port"), c.GlobalString("poll-interval"), c.GlobalString("k8s"))
	}
}
