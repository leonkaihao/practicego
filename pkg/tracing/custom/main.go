package main

import (
	"context"
	"fmt"
	"practicego/pkg/tracing/util"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type MD map[string][]string

func (c MD) ForeachKey(handler func(key, val string) error) error {
	for k, v := range c {
		vals := ""
		for i, d := range v {
			if i == 0 {
				vals += d
			} else {
				vals += ("," + d)
			}
		}
		if err := handler(k, vals); err != nil {
			return err
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
func (c MD) Set(key, val string) {
	c[key] = strings.Split(val, ",")
}

type dataex struct {
	A        int
	B        float64
	C        string
	metadata MD
}

type payloadKey struct{}

// create context with pipeline
func createClient(ctx context.Context) context.Context {
	pl := make(chan *dataex, 15)
	ctx = context.WithValue(ctx, payloadKey{}, pl)

	span, ctx := opentracing.StartSpanFromContext(ctx, "createClient")
	datas := []*dataex{
		{A: 1, B: 2.0, C: "3"},
		{A: 4, B: 5.0, C: "6"},
		{A: 7, B: 8.0, C: "9"},
	}
	go func() {
		span.Finish()
		defer close(pl)
		for i, data := range datas {
			if data.metadata == nil {
				data.metadata = make(map[string][]string)
			}
			subspan, _ := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("send-%v", i))
			opentracing.GlobalTracer().Inject(
				subspan.Context(),
				opentracing.TextMap,
				data.metadata)
			pl <- data
			subspan.Finish()
		}
	}()

	return ctx
}

// receive span from pipeline
func createServer(ctx context.Context) {
	pl := ctx.Value(payloadKey{}).(chan *dataex)
	count := 0
	for data := range pl {
		sctx, err := opentracing.GlobalTracer().Extract(
			opentracing.TextMap, MD(data.metadata))
		if err != nil {
			continue
		}
		span := opentracing.StartSpan(
			fmt.Sprintf("recv-%v-item", count),
			ext.RPCServerOption(sctx))
		ctx = opentracing.ContextWithSpan(ctx, span)
		count++
		span.Finish()
	}
}

func main() {

	closer := util.InitJaeger("custom")
	defer closer.Close()
	t := opentracing.GlobalTracer()
	span := t.StartSpan("foobar")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)
	ctx = createClient(ctx)
	createServer(ctx)
}
