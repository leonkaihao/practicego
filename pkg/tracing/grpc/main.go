package main

import (
	"context"
	"log"
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"

	"practicego/pkg/tracing/util"
)

const (
	port        = ":50051"
	address     = "localhost:50051"
	defaultName = "world"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "clientsayhello")
	defer span.Finish()
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		span.SetTag("foo", md.Get("foo")[0])
	}
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func createServer() {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())))
	pb.RegisterGreeterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func createClient(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "createClient")
	defer span.Finish()
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	md := metadata.MD{}
	md.Set("foo", "bar")
	ctx = metadata.NewOutgoingContext(ctx, md)
	// Contact the server and print out its response.
	name := defaultName

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}

func main() {

	closer := util.InitJaeger("basic")
	defer closer.Close()
	t := opentracing.GlobalTracer()
	span := t.StartSpan("foobar")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)

	go createServer()
	createClient(ctx)
}
