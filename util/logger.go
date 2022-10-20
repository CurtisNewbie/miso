package util

import "github.com/sirupsen/logrus"

// Get pre-configured TextFormatter for logrus
func PreConfiguredFormatter() *logrus.TextFormatter {
	return &logrus.TextFormatter{
		FullTimestamp: true,
	}
}
