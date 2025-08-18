package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	env string
	log zerolog.Logger
}

func New(env string) *Logger {
	var zl zerolog.Logger

	// Configure zerolog for dev vs prod
	if env == "dev" {
		// Human-friendly console output
		zl = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	} else {
		// JSON structured logs for production
		zl = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return &Logger{
		env: env,
		log: zl,
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Info().Msgf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Error().Msgf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatal().Msgf(format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		env: l.env,
		log: l.log.With().Interface(key, value).Logger(),
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.log.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{
		env: l.env,
		log: ctx.Logger(),
	}
}
