package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"practicego/pkg/tracing/util"

	"github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
)

func copyOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

func createCaller(ctx context.Context) error {
	sdata := opentracing.TextMapCarrier{}
	subspan, _ := opentracing.StartSpanFromContext(ctx, "createCaller")
	defer subspan.Finish()

	// pass span through command args
	opentracing.GlobalTracer().Inject(
		subspan.Context(),
		opentracing.TextMap,
		sdata)
	spanStr, err := json.Marshal(sdata)
	if err != nil {
		return err
	}
	cmd := exec.Command("python", "pkg/tracing/testcode/python/console.py", "-s", string(spanStr))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go copyOutput(stdout)
	go copyOutput(stderr)
	err = cmd.Wait()
	return err
}

func main() {

	closer := util.InitJaeger("console")
	defer closer.Close()
	t := opentracing.GlobalTracer()
	span := t.StartSpan("foobar")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)
	err := createCaller(ctx)
	if err != nil {
		log.Error(err)
	}
}
