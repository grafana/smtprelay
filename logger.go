package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func setupLogger(logFile, logLevel string) (*logrus.Entry, error) {
	writer, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot open log file %q: %w", logFile, err)
	}

	logger := logrus.New()

	logger.SetOutput(writer)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat:   time.RFC3339Nano,
		DisableHTMLEscape: true,
	})

	logEntry := logrus.NewEntry(logger)

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.DebugLevel

		logEntry.WithField("given_level", logLevel).
			Warn("could not parse log level, defaulting to 'debug'")
	}

	logger.SetLevel(level)

	return logEntry, nil
}
