package worker

import (
	"github.com/rs/zerolog/log"
)

type Logger interface {
	Debug(args ...interface{})

	Info(args ...interface{})

	Warn(args ...interface{})

	Error(args ...interface{})

	Fatal(args ...interface{})
}

type WorkLogger struct {
}

func NewLogger() *WorkLogger {
	return &WorkLogger{}
}

func (logger WorkLogger) Debug(args ...interface{}) {
	log.Debug().Msgf("%v", args)
}
func (logger WorkLogger) Info(args ...interface{}) {
	log.Info().Msgf("%v", args)
}
func (logger WorkLogger) Warn(args ...interface{}) {
	log.Warn().Msgf("%v", args)
}
func (logger WorkLogger) Error(args ...interface{}) {
	log.Error().Msgf("%v", args)
}
func (logger WorkLogger) Fatal(args ...interface{}) {
	log.Fatal().Msgf("%v", args)
}
