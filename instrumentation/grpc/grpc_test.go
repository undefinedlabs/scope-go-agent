package grpc

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go/ext"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/tracer"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	port        = ":50051"
	address     = "localhost:50051"
	defaultName = "world"
)

var client pb.GreeterClient
var r *tracer.InMemorySpanRecorder

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
func (s *server) SayHelloAgain(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Llongfile)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer(GetServerInterceptors()...)
	pb.RegisterGreeterServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	opts := append([]grpc.DialOption{grpc.WithInsecure()}, GetClientInterceptors()...)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client = pb.NewGreeterClient(conn)

	// Test tracer
	r = tracer.NewInMemoryRecorder()
	instrumentation.SetTracer(tracer.New(r))

	os.Exit(m.Run())
}

func TestGrpc(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ret, err := client.SayHello(ctx, &pb.HelloRequest{Name: defaultName})
	if err != nil {
		t.Fatalf("could not greet: %v", err)
	}
	t.Logf("Greeting: %s", ret.GetMessage())

	spans := r.GetSpans()

	if len(spans) != 2 {
		t.Fatalf("there aren't the right number of spans: %d", len(spans))
	}

	serverSpan := spans[0]
	if serverSpan.Tags[string(ext.SpanKind)] != ext.SpanKindRPCServerEnum {
		t.Fatalf("the server span doesn't have the right span kind")
	}
	if serverSpan.Tags[Status] != "OK" {
		t.Fatalf("the server span doesn't have the right status")
	}

	clientSpan := spans[1]
	if clientSpan.Tags[string(ext.SpanKind)] != ext.SpanKindRPCClientEnum {
		t.Fatalf("the client span doesn't have the right span kind")
	}
	if clientSpan.Tags[Status] != "OK" {
		t.Fatalf("the client span doesn't have the right status")
	}
}
