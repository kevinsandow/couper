//go:generate go run assets/generate/generate.go

package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"go.avenga.cloud/couper/gateway/command"
	"go.avenga.cloud/couper/gateway/config"
	"go.avenga.cloud/couper/gateway/server"
)

const defaultConfigFile = "example.hcl"

var (
	configFile = flag.String("f", defaultConfigFile, "-f ./couper.conf")
)

func main() {
	// TODO: command / args
	if !flag.Parsed() {
		flag.Parse()
	}

	logger := newLogger()

	exampleConf := config.LoadFile(*configFile, logger)

	err := os.Chdir(filepath.Dir(*configFile))
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	exampleConf.WD = wd

	ctx := command.ContextWithSignal(context.Background())
	srv := server.New(ctx, logger, exampleConf)
	srv.Listen()
	<-ctx.Done() // TODO: shutdown deadline
}

func newLogger() *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
		logrus.FieldKeyTime: "timestamp",
		logrus.FieldKeyMsg:  "message",
	}}
	logger.Level = logrus.DebugLevel
	return logger.WithField("type", "couper")
}
