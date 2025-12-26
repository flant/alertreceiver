package logging

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type Logger struct{}

var initOnce sync.Once

func NewLogger() *Logger {
	initOnce.Do(func() {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetLevel(log.InfoLevel)
	})
	return &Logger{}
}

func (l *Logger) Info(message string, fields log.Fields) {
	log.WithFields(fields).Info(message)
}

func (l *Logger) Warn(message string, fields log.Fields) {
	log.WithFields(fields).Warn(message)
}

func (l *Logger) Error(message string, fields log.Fields) {
	log.WithFields(fields).Error(message)
}
