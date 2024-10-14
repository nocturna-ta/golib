package grpc

import (
	"context"
	"fmt"
	"github.com/newrelic/go-agent/v3/integrations/nrgrpc"
	"github.com/nocturna-ta/golib/log"
	newrelicLib "github.com/nocturna-ta/golib/tracing/newrelic"
	sentryLib "github.com/nocturna-ta/golib/tracing/sentry"
	"github.com/valyala/fasthttp/reuseport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"os"
	"reflect"
	"syscall"
)

type ServerOptions struct {
	Port                         uint
	UnaryInterceptors            []grpc.UnaryServerInterceptor
	StreamServerInterceptors     []grpc.StreamServerInterceptor
	CertFile                     string
	KeyFile                      string
	SkipRegisterReflectionServer bool
	NewRelicOpts                 *newrelicLib.Options
	SentryConfig                 *sentryLib.Config
}

type Server struct {
	s    *grpc.Server
	port int
}

func NewServer(serverOpts *ServerOptions) *Server {
	nrApp := newrelicLib.SetupNewRelic(serverOpts.NewRelicOpts)

	if serverOpts.SentryConfig != nil {
		_ = sentryLib.Init(serverOpts.SentryConfig)
	}

	unaryServerInterceptors := append([]grpc.UnaryServerInterceptor{
		UnaryServerPanicInterceptor(),
		SentryUnaryServerInterceptor(),
		RequestContextServerInterceptor(),
		nrgrpc.UnaryServerInterceptor(nrApp),
	}, serverOpts.UnaryInterceptors...)

	streamServerInterceptors := append([]grpc.StreamServerInterceptor{
		nrgrpc.StreamServerInterceptor(nrApp),
	}, serverOpts.StreamServerInterceptors...)

	var opts []grpc.ServerOption

	if serverOpts.CertFile != "" && serverOpts.KeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(serverOpts.CertFile, serverOpts.KeyFile)

		if err != nil {
			log.Fatalf("Failed loading certificates: %v\n", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	opts = append(opts,
		grpc.ChainUnaryInterceptor(unaryServerInterceptors...),
		grpc.ChainStreamInterceptor(streamServerInterceptors...))

	srv := grpc.NewServer(opts...)

	if !serverOpts.SkipRegisterReflectionServer {
		reflection.Register(srv)
	}

	return &Server{
		s:    srv,
		port: int(serverOpts.Port),
	}
}

func (gs *Server) Register(fn any, server any) {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		log.Fatal("first parameter must be a function")
	}

	vargs := make([]reflect.Value, 2)
	vargs[0] = reflect.ValueOf(gs.s)
	vargs[1] = reflect.ValueOf(server)
	v.Call(vargs)
}

func (gs *Server) Start() error {
	address := fmt.Sprintf(":%d", gs.port)
	l, err := reuseport.Listen("tcp4", address)
	if err != nil {
		return err
	}

	log.Print("starting grpc server on ", address)
	return gs.s.Serve(l)
}

func (gs *Server) MustStart() {
	err := gs.Start()
	if err != nil {
		log.Fatalf("failed to start grpc server: %v", err)
	}
}

func (gs *Server) CatchSignal(s os.Signal) {
	log.Print("grpc service got signal:", s)
	if s == syscall.SIGHUP || s == syscall.SIGTERM {
		gs.Stop()
	}
}

func (gs *Server) Stop() {
	gs.StopContext(context.Background())
}

func (gs *Server) StopContext(ctx context.Context) {
	done := make(chan bool)

	go func(doneCh chan<- bool) {
		gs.s.GracefulStop()
		doneCh <- true
	}(done)

	select {
	case <-ctx.Done():
		log.Warn("timeout waiting grpc server to stopped...")
	case <-done:
		close(done)
		log.Info("grpc server stopped...")
	}
}
