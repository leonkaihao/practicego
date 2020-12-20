package contexttracer

import (
	"context"
	"io"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type tracerContextKey struct{}

// TraceInfo includes the tracing context items
// It is expected to be updated and passed through the context chain.
// It is inserted to context by withTraceInfo or NewContext
// A copy of it can be got by fromContext
// It is updated by WithInheritableSpanTags or StartSpanFromContext
type TracerInfo struct {
	Tracer opentracing.Tracer
	Closer io.Closer
}

func NewContext(ctx context.Context, tracer opentracing.Tracer, closer io.Closer) context.Context {
	return context.WithValue(ctx, tracerContextKey{}, TracerInfo{tracer, closer})
}

func FromContext(ctx context.Context) (tracerInfo TracerInfo, ok bool) {
	tracerInfo, ok = ctx.Value(tracerContextKey{}).(TracerInfo)
	return
}

func FromContextOrGlobal(ctx context.Context) opentracing.Tracer {
	tracerInfo, ok := FromContext(ctx)
	if !ok || tracerInfo.Tracer == nil {
		return opentracing.GlobalTracer()
	}
	return tracerInfo.Tracer
}

// Simlar to opentracing.StartSpanFromContext but this uses the tracer stored in the context instead of the global tracer.
// Based on opentracing.startSpanFromContextWithTracer.
func StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	tracer := FromContextOrGlobal(ctx)
	var span opentracing.Span
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		opts = append(opts, opentracing.ChildOf(parentSpan.Context()))
	}
	if tags := inheritableTagsCopyFromContext(ctx); len(tags) != 0 {
		ctx = WithInheritableSpanTags(ctx, tags)
		opts = append(opts, tags)
	}
	span = tracer.StartSpan(operationName, opts...)

	return span, opentracing.ContextWithSpan(ctx, span)
}

// StartSpanFromHttpRequest extracts the opentracing span context from the HTTP request and creates
// a new span with that span context.
// See also StartSpanFromContext.
func StartSpanFromHttpRequest(ctx context.Context, req *http.Request, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	tracer := FromContextOrGlobal(ctx)
	wireContext, err := tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))
	if err != nil {
		// This will occur if called from a web browser or curl.
		log.Debug(errors.Wrap(err, "contexttracer.StartSpanFromHttpRequest"))
	}
	if tags := inheritableTagsCopyFromContext(ctx); len(tags) != 0 {
		opts = append(opts, tags)
	}
	span := tracer.StartSpan(
		operationName,
		append([]opentracing.StartSpanOption{ext.RPCServerOption(wireContext)}, opts...)...)
	return span, opentracing.ContextWithSpan(ctx, span)
}

func Close(ctx context.Context) error {
	tracerInfo, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	if tracerInfo.Closer == nil {
		return nil
	}
	return tracerInfo.Closer.Close()
}

type inheritableSpanTagsKey struct{}

func inheritableTagsCopyFromContext(ctx context.Context) opentracing.Tags {
	tags := opentracing.Tags{}
	if it, ok := ctx.Value(inheritableSpanTagsKey{}).(opentracing.Tags); ok {
		for k, v := range it {
			tags[k] = v
		}
	}
	return tags
}

// WithInheritableSpanTags creates context with updating inheritable tags
func WithInheritableSpanTags(ctx context.Context, newTags opentracing.Tags) context.Context {
	tags := inheritableTagsCopyFromContext(ctx)
	for k, v := range newTags {
		tags[k] = v
	}
	return context.WithValue(ctx, inheritableSpanTagsKey{}, tags)
}
