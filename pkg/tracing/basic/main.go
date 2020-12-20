package main

import (
	"context"
	"errors"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"practicego/pkg/tracing/util"
)

func main() {
	closer := util.InitJaeger("basic")
	defer closer.Close()
	t := opentracing.GlobalTracer()
	span := t.StartSpan("foobar")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)
	Foo(ctx)
}

func Foo(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Foo")
	defer span.Finish()
	span.SetTag("foo", "yes")
	span.SetBaggageItem("creator", "foo")

	Bar(ctx)
}

func Bar(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Bar")
	defer span.Finish()

	span.SetTag("bar", "yes")
	span.SetTag("creator", span.BaggageItem("creator"))

	err := errors.New("something wrong")
	span.LogFields(
		log.String("event", "error"),
		log.String("message", err.Error()),
	)
	span.SetTag("error", true)
}
