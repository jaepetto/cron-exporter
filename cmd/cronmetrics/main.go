package main

import (
	"os"

	"github.com/jaep/cron-exporter/internal/cli"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cli.Execute(); err != nil {
		logrus.WithError(err).Fatal("command failed")
		os.Exit(1)
	}
}
