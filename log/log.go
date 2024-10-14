package log

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golib/custerr"
	"os"
	"runtime"
	"strings"
	"time"
)

type Fields map[string]any

func init() {
	log.Logger = log.Logger.Hook(TracingHook{})
	SetLevel("info")
}

func SetLevel(level string) {
	switch level {
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "disabled":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func SetFormatter(format string) {
	switch format {
	case "json":
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	case "text":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC822})
	default:
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
	log.Logger = log.Logger.Hook(TracingHook{})
}

func Info(args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Info().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Infof(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Info().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func InfoWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)

	log.Info().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Print(args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Info().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Printf(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Info().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func PrintWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)

	log.Info().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Debug(args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Debug().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Debugf(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Debug().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func DebugWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)

	log.Debug().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Warn(args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Warn().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Warnf(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	log.Warn().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func WarnWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)

	log.Warn().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Error(args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	log.Error().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Errorf(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(fmt.Errorf(format, args...))
	log.Error().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func ErrorWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	log.Error().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Fatal(args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	log.Fatal().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func Fatalf(format string, args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(fmt.Errorf(format, args...))
	log.Fatal().
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprintf(format, args...))
}

func FatalWithCtx(ctx context.Context, args ...any) {
	file, line, _ := getFileLinePc(2)
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	log.Fatal().
		Ctx(ctx).
		Str("source", fmt.Sprintf("%s:%d", file, line)).
		Msg(fmt.Sprint(args...))
}

func WithFields(fields Fields) *Entry {
	file, line, pc := getFileLinePc(2)

	// add field for function name
	funcname := runtime.FuncForPC(pc).Name()
	fn := funcname[strings.LastIndex(funcname, ".")+1:]

	logFields := Fields{
		"source":   fmt.Sprintf("%s:%d", file, line),
		"function": fn,
	}

	for key, value := range fields {
		logFields[key] = value
		if err := getError(value); err != nil {
			sentry.CaptureException(err)
		}
	}

	return &Entry{
		logger: log.Logger,
		fields: logFields,
	}
}

func WithError(err error) *Entry {
	file, line, _ := getFileLinePc(2)

	fields := Fields{
		"source": fmt.Sprintf("%s:%d", file, line),
		"error":  err,
	}
	sentry.CaptureException(err)

	return &Entry{
		logger: log.Logger,
		fields: fields,
	}
}

func getFileLinePc(skip int) (string, int, uintptr) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		file = "<???>"
		line = 1
	}

	return file, line, pc
}

func getError(entry any) error {
	err, ok := entry.(error)
	if !ok {
		return nil
	}
	_, ok = err.(*custerr.ErrChain)
	if !ok {
		return err
	}
	return nil
}
