package common

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(CustomFormatter())
}

type CTFormatter struct {
}

func (c *CTFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	s := fmt.Sprintf("[%s] %s - %s\n", strings.ToUpper(entry.Level.String()), entry.Time.Format("2006-01-02 15:04:05"), entry.Message)
	return []byte(s), nil
}

// Get custom formatter logrus
func CustomFormatter() logrus.Formatter {
	return &CTFormatter{}
}

// Get pre-configured TextFormatter for logrus
func PreConfiguredFormatter() logrus.Formatter {
	return &logrus.TextFormatter{
		FullTimestamp: true,
	}
}
