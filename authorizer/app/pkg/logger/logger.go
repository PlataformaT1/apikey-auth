package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	graylog "github.com/gemnasium/logrus-graylog-hook/v3"
	"github.com/sirupsen/logrus"
)

type Channel int

const (
	Stdout Channel = iota
	Stdgraylog
	Stdgrayout
)

func New() (*logrus.Logger, error) {
	var (
		logChannel  = os.Getenv("USER_VAR_LOG_CHAN")
		logLevel    = os.Getenv("USER_VAR_LOG_LEVEL")
		graylogAddr = os.Getenv("GRAYLOG_ADDR")
		formatter   = &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			DisableColors:   false,
			CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
				function, file := "", fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
				return function, file
			},
		}
	)
	channel, err := ParseChannel(logChannel)
	if err != nil {
		return nil, err
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	log := logrus.New()
	log.SetFormatter(formatter)
	log.SetReportCaller(true)
	log.SetLevel(level)

	switch channel {
	case Stdout:
		log.SetOutput(os.Stdout)
	case Stdgraylog:
		hook := graylog.NewGraylogHook(graylogAddr, nil)
		log.AddHook(hook)
		log.SetOutput(io.Discard)
	case Stdgrayout:
		hook := graylog.NewGraylogHook(graylogAddr, nil)
		log.AddHook(hook)
		log.SetOutput(os.Stdout)
	}

	return log, nil
}

func ParseChannel(channel string) (Channel, error) {
	switch strings.ToLower(channel) {
	case "stdout":
		return Stdout, nil
	case "stdgraylog":
		return Stdgraylog, nil
	case "stdgrayout":
		return Stdgrayout, nil
	}
	var ch Channel
	return ch, fmt.Errorf("not a valid channel: %s", channel)
}
