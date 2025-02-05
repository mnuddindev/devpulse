package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

// start logger
func StartLogger() {
	// config logger
	Log.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: false,
		PrettyPrint:      true,
		TimestampFormat:  "2025-02-05 6:30:25",
	})
	Log.SetOutput(os.Stdout)

	// log to file
	file, err := os.OpenFile("devpulse.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.SetOutput(file)
	} else {
		Log.Info("Failed to log to file, using default stderr")
	}

	Log.SetLevel(logrus.InfoLevel)
}
