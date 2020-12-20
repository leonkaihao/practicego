package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"practicego/pkg/tracing/util"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func createServer() {
	closer := util.InitJaeger("http")
	defer closer.Close()
	t := opentracing.GlobalTracer()

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("got request")
		wireContext, err := t.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header))
		span := opentracing.StartSpan("http server test", ext.RPCServerOption(wireContext))
		defer span.Finish()
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		io.WriteString(w, "Hello, world!\n")
	}

	http.HandleFunc("/hello", helloHandler)
	log.Fatal(http.ListenAndServe(":1234", nil))
}

func clientCall() error {
	closer := util.InitJaeger("http")
	defer closer.Close()
	t := opentracing.GlobalTracer()
	span := t.StartSpan("http client test")
	defer span.Finish()

	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://localhost:1234/hello", nil)
	if err != nil {
		return err
	}
	err = t.Inject(span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	return err
}

func main() {
	go createServer()
	clientCall()
}
