package tracing

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type nrTransactionKey struct{}

var (
	NewRelicTransactionKey = nrTransactionKey{}
)

type (
	span struct {
		nrSegment  *newrelic.Segment
		sentrySpan *sentry.Span
	}
)

type (
	SpanTrace interface {
		End()
	}
)

func StartSpanFromContext(ctx context.Context, spanName string) (SpanTrace, context.Context) {
	spanTrace := &span{}

	nrTxnVal := ctx.Value(NewRelicTransactionKey)
	if nrTxnVal != nil {
		nrTxn, ok := nrTxnVal.(*newrelic.Transaction)
		if ok {
			segment := nrTxn.StartSegment(spanName)
			spanTrace.nrSegment = segment
		}
	}

	spanTrace.sentrySpan = sentry.StartSpan(ctx, spanName, sentry.OpName(spanName))
	ctx = spanTrace.sentrySpan.Context()

	return spanTrace, ctx
}

func (s *span) End() {
	// end new relic segment
	if s.nrSegment != nil {
		s.nrSegment.End()
	}

	// end sentry span
	if s.sentrySpan != nil {
		s.sentrySpan.Finish()
	}
}
