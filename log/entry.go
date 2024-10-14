package log

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

type Entry struct {
	logger zerolog.Logger
	fields map[string]interface{}
}

func (e *Entry) Info(args ...any) {
	e.logger.Info().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Infof(format string, args ...any) {
	e.logger.Info().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) InfoWithCtx(ctx context.Context, args ...any) {
	e.logger.Info().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}

func (e *Entry) Print(args ...any) {
	e.logger.Info().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Printf(format string, args ...any) {
	e.logger.Info().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) PrintWithCtx(ctx context.Context, args ...any) {
	e.logger.Info().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}

func (e *Entry) Debug(args ...any) {
	e.logger.Debug().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Debugf(format string, args ...any) {
	e.logger.Debug().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) DebugWithCtx(ctx context.Context, args ...any) {
	e.logger.Debug().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}

func (e *Entry) Warn(args ...any) {
	e.logger.Warn().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Warnf(format string, args ...any) {
	e.logger.Warn().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) WarnWithCtx(ctx context.Context, args ...any) {
	e.logger.Warn().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}

func (e *Entry) Error(args ...any) {
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	e.logger.Error().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Errorf(format string, args ...any) {
	sentry.CaptureException(fmt.Errorf(format, args...))
	e.logger.Error().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) ErrorWithCtx(ctx context.Context, args ...any) {
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	e.logger.Error().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}

func (e *Entry) Fatal(args ...any) {
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	e.logger.Fatal().Fields(e.fields).Msg(fmt.Sprint(args...))
}

func (e *Entry) Fatalf(format string, args ...any) {
	sentry.CaptureException(fmt.Errorf(format, args...))
	e.logger.Fatal().Fields(e.fields).Msg(fmt.Sprintf(format, args...))
}

func (e *Entry) FatalWithCtx(ctx context.Context, args ...any) {
	sentry.CaptureException(errors.New(fmt.Sprint(args...)))
	e.logger.Fatal().Fields(e.fields).Ctx(ctx).Msg(fmt.Sprint(args...))
}
